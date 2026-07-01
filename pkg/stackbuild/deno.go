package stackbuild

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/moby/buildkit/client/llb"
	"miren.dev/runtime/pkg/imagerefs"
)

// denoEnvPatterns match Deno's Deno.env.get("VAR") accessor. Unlike
// process.env.VAR / Bun.env.VAR (property access), Deno.env.get is a
// function call, so it needs its own pattern rather than reusing
// nodeEnvPatterns/bunEnvPatterns.
var denoEnvPatterns = []*regexp.Regexp{
	// Deno.env.get("VAR") or Deno.env.get('VAR')
	regexp.MustCompile(`Deno\.env\.get\(['"]([A-Z][A-Z0-9_]+)['"]\)`),
}

// denoOptionalEnvPatterns detect Deno.env.get("VAR") usages with a fallback.
var denoOptionalEnvPatterns = []*regexp.Regexp{
	// Deno.env.get("VAR") ?? 'default'
	regexp.MustCompile(`Deno\.env\.get\(['"]([A-Z][A-Z0-9_]+)['"]\)\s*\?\?`),
	// Deno.env.get("VAR") || 'default'
	regexp.MustCompile(`Deno\.env\.get\(['"]([A-Z][A-Z0-9_]+)['"]\)\s*\|\|`),
}

// denoCommandRe matches a Deno subcommand invocation in a package.json
// script. Stricter than bun's bare `bunx?` match — bare "deno" would
// false-positive far more readily, so a known subcommand is required.
var denoCommandRe = regexp.MustCompile(`(?:^|\s)deno\s+(?:run|task|serve|start)(?:\s|$)`)

// denoJSONCCommentRe strips // line comments and /* */ block comments from
// deno.jsonc so it can be parsed with encoding/json. This is a best-effort
// normalization (not a full JSONC tokenizer), consistent with this
// package's existing lightweight parsing style.
var denoJSONCCommentRe = regexp.MustCompile(`//[^\n]*|/\*.*?\*/`)

// DenoStack implements Stack for Deno
type DenoStack struct {
	MetaStack

	// Detection state set in Init()
	configPath string // "deno.json" or "deno.jsonc", whichever was found
	tasks      map[string]string
	entryPoint string

	// Detected environment variable requirements
	requiredEnvVars []EnvVarRequirement
}

func (s *DenoStack) BaseDistro() string {
	return "debian"
}

func (s *DenoStack) Name() string {
	return "deno"
}

func (s *DenoStack) Detect() bool {
	if s.hasFile("deno.json") {
		s.Event("file", "deno.json", "Found deno.json")
		return true
	}
	if s.hasFile("deno.jsonc") {
		s.Event("file", "deno.jsonc", "Found deno.jsonc")
		return true
	}
	if s.hasFile("deno.lock") {
		s.Event("file", "deno.lock", "Found deno.lock (Deno runtime)")
		return true
	}
	if s.detectInFile("Procfile", `web:\s+deno`) {
		s.Event("file", "Procfile", "Procfile references deno")
		return true
	}
	if s.hasFile("package.json") && s.detectDenoInScripts() {
		s.Event("config", "scripts", "package.json scripts reference deno")
		return true
	}
	return false
}

func (s *DenoStack) detectDenoInScripts() bool {
	scripts := s.readPackageScripts()
	for _, cmd := range scripts {
		if denoCommandRe.MatchString(cmd) {
			return true
		}
	}
	return false
}

// readPackageScripts reads only the scripts section of package.json.
// Used during Detect() before Init() runs parseDenoConfig.
func (s *DenoStack) readPackageScripts() map[string]string {
	data, err := s.readFile("package.json")
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	return pkg.Scripts
}

func (s *DenoStack) Init(opts BuildOptions) {
	s.SetCwd("/app")

	s.parseDenoConfig()

	if s.tasks != nil {
		if _, ok := s.tasks["start"]; ok {
			s.Event("script", "start", "deno.json start task detected")
		}
	}

	// Check for common entry points and store the first one found.
	// Deno idioms (main.ts/mod.ts) are checked before the Node/Bun-style
	// index.ts/server.ts/app.ts names, since Deno-first projects
	// overwhelmingly favor them, and .ts before .js for each name since
	// Deno projects are overwhelmingly TypeScript.
	for _, entry := range []string{"main.ts", "mod.ts", "index.ts", "server.ts", "app.ts", "main.js", "mod.js", "index.js", "server.js", "app.js"} {
		if s.hasFile(entry) {
			s.entryPoint = entry
			s.Event("file", entry, "Entry point file detected")
			break
		}
	}

	// Detect required environment variables
	s.requiredEnvVars = s.detectEnvVars()
	for _, ev := range s.requiredEnvVars {
		s.Event("env_var", ev.Name, ev.Reason)
	}
}

// parseDenoConfig reads deno.json (or deno.jsonc, comments stripped) and
// records its "tasks" map.
func (s *DenoStack) parseDenoConfig() {
	for _, name := range []string{"deno.json", "deno.jsonc"} {
		data, err := s.readFile(name)
		if err != nil {
			continue
		}
		if strings.HasSuffix(name, ".jsonc") {
			data = denoJSONCCommentRe.ReplaceAll(data, nil)
		}
		var cfg struct {
			Tasks map[string]string `json:"tasks"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		s.configPath = name
		s.tasks = cfg.Tasks
		return
	}
}

func (s *DenoStack) GenerateLLB(dir string, opts BuildOptions) (*llb.State, error) {
	// Set up local context with the directory
	localCtx := llb.Local("context",
		llb.SharedKeyHint(dir),
		llb.ExcludePatterns([]string{".git"}),
		llb.FollowPaths([]string{"."}),
		llb.WithCustomName("application code"),
	)

	version := "2"
	if opts.Version != "" {
		version = opts.Version
	}
	base := llb.Image(imagerefs.GetDenoImage(version))

	base = s.addAppUser(base)

	// Copy config/lock files first for better caching.
	cfgFiles := []string{"deno.json", "deno.jsonc", "deno.lock", "import_map.json"}
	depState := base.File(llb.Copy(localCtx, "/", "/app", &llb.CopyInfo{
		IncludePatterns:    cfgFiles,
		CreateDestPath:     true,
		AllowWildcard:      true,
		AllowEmptyWildcard: true,
	}), llb.WithCustomName("copy deno config files"))

	// Persistent Deno dependency cache. DENO_DIR is scoped to this Run op
	// only (not persisted via MetaStack.AddEnv) since it's purely a
	// build-time cache location, not something the running app needs.
	denoCache := llb.Scratch().File(
		llb.Mkdir("/deno-cache", 0755, llb.WithParents(true)),
	)

	state := depState.Dir("/app").Run(
		llb.Shlex(s.cacheWarmCommand()),
		llb.AddEnv("DENO_DIR", "/deno-cache"),
		llb.AddMount("/deno-cache", denoCache, llb.AsPersistentCacheDir("deno", llb.CacheMountShared)),
		llb.WithCustomName("[phase] Installing Deno dependencies"),
	).Root()

	h := &highlevelBuilder{opts}

	// Copy the rest of the application code
	state = h.copyApp(state, localCtx)

	state = s.applyOnBuild(state, opts)

	return &state, nil
}

// cacheWarmCommand resolves and caches dependencies (JSR, npm, and URL
// imports) ahead of copying in the full application code. deno install
// against a specific entrypoint also resolves that file's imports; with no
// entrypoint, a bare "deno install" still resolves deno.json's "imports"
// map (and is a safe no-op if there's nothing to resolve).
func (s *DenoStack) cacheWarmCommand() string {
	if s.entryPoint != "" {
		return "deno install --entrypoint " + s.entryPoint
	}
	return "deno install"
}

func (s *DenoStack) WebCommand() string {
	// A deno.json task is assumed to already carry whatever permission
	// flags the app needs, so prefer it over synthesizing a command.
	if s.tasks != nil {
		for _, task := range []string{"start", "serve", "server"} {
			if _, ok := s.tasks[task]; ok {
				return "deno task " + task
			}
		}
	}

	// Fallback: run the entry point directly. -A (allow-all) is used
	// because guessing a minimal permission set without static analysis
	// risks under-provisioning and a runtime permission-denial crash,
	// which is worse than the security-permissive default here.
	if s.entryPoint != "" {
		return "deno run -A " + s.entryPoint
	}

	return ""
}

// RequiredEnvVars returns the detected environment variable requirements
func (s *DenoStack) RequiredEnvVars() []EnvVarRequirement {
	return s.requiredEnvVars
}

// detectEnvVars analyzes the app to find required environment variables
func (s *DenoStack) detectEnvVars() []EnvVarRequirement {
	var results []EnvVarRequirement

	// Scan source code for env var usage. Deno.env.get(...) is combined
	// with the Node patterns since Deno polyfills process.env for
	// npm-compat code.
	envPatterns := append(append([]*regexp.Regexp{}, nodeEnvPatterns...), denoEnvPatterns...)
	optionalPatterns := append(append([]*regexp.Regexp{}, nodeOptionalEnvPatterns...), denoOptionalEnvPatterns...)
	sourceVars := scanSourceFilesForEnvVars(s.dir, []string{".ts", ".tsx", ".js", ".jsx"}, envPatterns, optionalPatterns)

	// Add source-detected vars. Direct, non-default code references are
	// hard requirements, so default to "required" rather than the weaker
	// "recommended" used elsewhere for package-inferred guesses (Deno has
	// no analogous package-name-keyed inference map — see node.go's
	// nodePackageEnvVars, which doesn't key cleanly against Deno's
	// versioned npm:/jsr: specifiers).
	for _, v := range sourceVars {
		if !hasEnvVar(results, v.name) {
			confidence := "required"
			reason := "Referenced in application code"
			if v.optional {
				confidence = "optional"
				reason = "Referenced in application code (has default)"
			}
			results = append(results, EnvVarRequirement{
				Name:       v.name,
				Source:     "code",
				Confidence: confidence,
				Reason:     reason,
			})
		}
	}

	// Config file parsing (.env.sample, .env.example)
	for _, filename := range []string{".env.sample", ".env.example"} {
		sampleVars := parseEnvSampleFile(s.dir, filename)
		for _, v := range sampleVars {
			if !hasEnvVar(results, v) {
				results = append(results, EnvVarRequirement{
					Name:       v,
					Source:     "config",
					Confidence: "required",
					Reason:     "Declared in " + filename,
				})
			}
		}
	}

	return results
}
