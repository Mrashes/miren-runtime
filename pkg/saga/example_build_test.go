// Example: Application Build & Deploy
//
// This example demonstrates realistic saga patterns using a simplified version
// of our build/deploy pipeline. Unlike the sandwich example, this one exercises
// the actual entity system (via in-memory server) and shows patterns you'd use
// in production code.
//
// Key concepts demonstrated:
//
//   - Entity integration: Actions create/update/delete real entities (App,
//     AppVersion, Artifact) using the in-memory entity server from testutils
//
//   - Non-serializable data: The SourceStager pattern shows how to handle
//     io.Reader streams that can't be serialized into the saga log. The stream
//     is registered before the saga starts, and the first action (StageSource)
//     extracts it to durable storage. See RFD-35 "Handling Non-Serializable Data".
//
//   - Find-or-create with conditional undo: ResolveArtifact looks up an existing
//     artifact by digest before creating one. The output includes a "Created" flag
//     so UndoResolveArtifact knows whether to delete (we created it) or skip
//     (it already existed).
//
//   - Previous state capture: ActivateVersion records the previous active version
//     in its output so UndoActivateVersion can restore it.
//
// The saga models these steps from servers/build/build.go:
//
//	StageSource     → Extract tar stream to durable storage
//	BuildImage      → Run BuildKit to produce container image
//	ResolveArtifact → Find or create Artifact entity by manifest digest
//	CreateVersion   → Create AppVersion entity with config
//	ActivateVersion → Set as App's active version
//
// Run the tests:
//
//	go test -v ./pkg/saga/... -run TestBuildSaga
//
// See example_sandwich_test.go for a simpler introduction to saga mechanics.
package saga_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	"miren.dev/runtime/api/core/core_v1alpha"
	apiserver "miren.dev/runtime/api/entityserver"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/entity/testutils"
	"miren.dev/runtime/pkg/saga"
)

// --- Stubbed Collaborators ---

// SourceStager handles non-serializable stream data at saga boundaries.
// This implements the pattern from the RFD for handling io.Reader streams:
//  1. Entry point registers stream with a generated ID
//  2. First action extracts stream to durable storage
//  3. On recovery, action checks for staged path instead of stream
type SourceStager struct {
	mu sync.Mutex

	// Active streams waiting to be staged (keyed by stream ID)
	// These are ephemeral - lost on crash, which is fine because
	// the saga will fail at StageSource and compensate
	streams map[string]io.Reader

	// Staged source directories (keyed by stream ID)
	// These are "durable" - in real impl would be on disk
	staged map[string]string

	// Track operations for testing
	events []string
}

func NewSourceStager() *SourceStager {
	return &SourceStager{
		streams: make(map[string]io.Reader),
		staged:  make(map[string]string),
	}
}

// RegisterStream registers a stream before starting a saga.
// Returns a stream ID that can be passed as saga input.
// The stream itself is NOT serializable - only the ID is.
func (s *SourceStager) RegisterStream(streamID string, r io.Reader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[streamID] = r
	s.events = append(s.events, fmt.Sprintf("Registered stream %s", streamID))
}

// Stage reads from the registered stream and "writes" to durable storage.
// Returns the path where source was staged.
// If the stream was already staged (recovery case), returns existing path.
func (s *SourceStager) Stage(streamID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already staged (recovery case)
	if path, ok := s.staged[streamID]; ok {
		s.events = append(s.events, fmt.Sprintf("Found existing staged source at %s", path))
		return path, nil
	}

	// Get the stream
	stream, ok := s.streams[streamID]
	if !ok {
		// Stream not found - either never registered or we crashed and lost it
		return "", fmt.Errorf("stream %s not found (may have been lost in crash)", streamID)
	}

	// "Read" from the stream (in real impl: extract tar to temp dir)
	// For stub, we just consume it and pretend we wrote it somewhere
	_, err := io.Copy(io.Discard, stream)
	if err != nil {
		return "", fmt.Errorf("failed to read stream: %w", err)
	}

	// Remove from active streams, add to staged
	delete(s.streams, streamID)
	path := fmt.Sprintf("/tmp/staged/%s", streamID)
	s.staged[streamID] = path

	s.events = append(s.events, fmt.Sprintf("Staged stream %s to %s", streamID, path))
	return path, nil
}

// Cleanup removes staged source (for undo)
func (s *SourceStager) Cleanup(streamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if path, ok := s.staged[streamID]; ok {
		delete(s.staged, streamID)
		s.events = append(s.events, fmt.Sprintf("Cleaned up staged source at %s", path))
	}
	return nil
}

// BuildKit is a stubbed build system that produces container images.
type BuildKit struct {
	// Configure the stub's behavior
	NextDigest   string
	NextImageURL string
	ShouldFail   bool
	FailMessage  string

	// Track what happened
	BuildCalled  bool
	BuiltFromDir string // Records which staged dir was used
}

func (bk *BuildKit) Build(appName, sourceDir string) (digest, imageURL string, err error) {
	bk.BuildCalled = true
	bk.BuiltFromDir = sourceDir
	if bk.ShouldFail {
		return "", "", errors.New(bk.FailMessage)
	}
	return bk.NextDigest, bk.NextImageURL, nil
}

// BuildLog tracks events during the build saga for observability.
type BuildLog struct {
	events []string
}

func (bl *BuildLog) record(event string) {
	bl.events = append(bl.events, event)
}

// --- Saga Actions ---

// StageSource extracts source from a non-serializable stream to durable storage.
// This is the critical first action that converts ephemeral stream data into
// a serializable path that can survive crashes.

type StageSourceIn struct {
	StreamID string
}

type StageSourceOut struct {
	SourceDir string
}

func StageSource(ctx context.Context, in StageSourceIn) (StageSourceOut, error) {
	stager := saga.Get[*SourceStager](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	sourceDir, err := stager.Stage(in.StreamID)
	if err != nil {
		buildLog.record(fmt.Sprintf("Failed to stage source: %v", err))
		return StageSourceOut{}, err
	}

	buildLog.record(fmt.Sprintf("Staged source to %s", sourceDir))
	return StageSourceOut{SourceDir: sourceDir}, nil
}

func UndoStageSource(ctx context.Context, in StageSourceIn, out StageSourceOut) error {
	stager := saga.Get[*SourceStager](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	if err := stager.Cleanup(in.StreamID); err != nil {
		return err
	}

	buildLog.record(fmt.Sprintf("Cleaned up staged source %s", out.SourceDir))
	return nil
}

// BuildImage calls BuildKit to produce a container image.
// Now takes SourceDir from StageSource instead of receiving stream directly.

type BuildImageIn struct {
	AppName   string
	SourceDir string // From StageSource - serializable!
}

type BuildImageOut struct {
	ManifestDigest string
	ImageURL       string
}

func BuildImage(ctx context.Context, in BuildImageIn) (BuildImageOut, error) {
	buildKit := saga.Get[*BuildKit](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	digest, imageURL, err := buildKit.Build(in.AppName, in.SourceDir)
	if err != nil {
		buildLog.record(fmt.Sprintf("Build failed: %v", err))
		return BuildImageOut{}, err
	}

	buildLog.record(fmt.Sprintf("Built image %s with digest %s", imageURL, digest))
	return BuildImageOut{
		ManifestDigest: digest,
		ImageURL:       imageURL,
	}, nil
}

func UndoBuildImage(ctx context.Context, in BuildImageIn, out BuildImageOut) error {
	buildLog := saga.Get[*BuildLog](ctx)
	buildLog.record(fmt.Sprintf("Build artifacts cleaned up for %s", in.AppName))
	return nil
}

// ResolveArtifact finds an existing artifact by digest or creates a new one.

type ResolveArtifactIn struct {
	AppName        string
	ManifestDigest string
}

type ResolveArtifactOut struct {
	ArtifactID string
	Created    bool // true if we created a new artifact (vs found existing)
}

func ResolveArtifact(ctx context.Context, in ResolveArtifactIn) (ResolveArtifactOut, error) {
	client := saga.Get[*apiserver.Client](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	// Try to find existing artifact by digest
	var existing core_v1alpha.Artifact
	err := client.OneAtIndex(ctx,
		entity.String(core_v1alpha.ArtifactManifestDigestId, in.ManifestDigest),
		&existing)

	if err == nil {
		buildLog.record(fmt.Sprintf("Found existing artifact %s for digest %s", existing.ID, in.ManifestDigest))
		return ResolveArtifactOut{
			ArtifactID: string(existing.ID),
			Created:    false,
		}, nil
	}

	// Look up app to get its ID for the artifact reference
	var app core_v1alpha.App
	if err := client.Get(ctx, in.AppName, &app); err != nil {
		return ResolveArtifactOut{}, fmt.Errorf("app not found: %w", err)
	}

	// Create new artifact
	artifact := &core_v1alpha.Artifact{
		App:            app.ID,
		ManifestDigest: in.ManifestDigest,
		Status:         core_v1alpha.ACTIVE,
	}

	artifactName := fmt.Sprintf("%s-%s", in.AppName, in.ManifestDigest[:12])
	id, err := client.Create(ctx, artifactName, artifact)
	if err != nil {
		return ResolveArtifactOut{}, fmt.Errorf("failed to create artifact: %w", err)
	}

	buildLog.record(fmt.Sprintf("Created artifact %s for digest %s", id, in.ManifestDigest))
	return ResolveArtifactOut{
		ArtifactID: string(id),
		Created:    true,
	}, nil
}

func UndoResolveArtifact(ctx context.Context, in ResolveArtifactIn, out ResolveArtifactOut) error {
	if !out.Created {
		// We didn't create it, nothing to undo
		return nil
	}

	client := saga.Get[*apiserver.Client](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	if err := client.Delete(ctx, entity.Id(out.ArtifactID)); err != nil {
		return fmt.Errorf("failed to delete artifact: %w", err)
	}

	buildLog.record(fmt.Sprintf("Deleted artifact %s", out.ArtifactID))
	return nil
}

// CreateVersion creates a new AppVersion entity.

type CreateVersionIn struct {
	AppName    string
	ArtifactID string
	ImageURL   string
}

type CreateVersionOut struct {
	VersionID string
}

func CreateVersion(ctx context.Context, in CreateVersionIn) (CreateVersionOut, error) {
	client := saga.Get[*apiserver.Client](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	// Look up app
	var app core_v1alpha.App
	if err := client.Get(ctx, in.AppName, &app); err != nil {
		return CreateVersionOut{}, fmt.Errorf("app not found: %w", err)
	}

	version := &core_v1alpha.AppVersion{
		App:      app.ID,
		Artifact: entity.Id(in.ArtifactID),
		ImageUrl: in.ImageURL,
		Version:  "v1", // simplified for example
		Config: core_v1alpha.Config{
			Commands: []core_v1alpha.Commands{
				{Service: "web", Command: "node server.js"},
			},
		},
	}

	versionName := fmt.Sprintf("%s-v1", in.AppName)
	id, err := client.Create(ctx, versionName, version)
	if err != nil {
		return CreateVersionOut{}, fmt.Errorf("failed to create version: %w", err)
	}

	buildLog.record(fmt.Sprintf("Created version %s", id))
	return CreateVersionOut{VersionID: string(id)}, nil
}

func UndoCreateVersion(ctx context.Context, in CreateVersionIn, out CreateVersionOut) error {
	client := saga.Get[*apiserver.Client](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	if err := client.Delete(ctx, entity.Id(out.VersionID)); err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	buildLog.record(fmt.Sprintf("Deleted version %s", out.VersionID))
	return nil
}

// ActivateVersion sets the new version as the app's active version.

type ActivateVersionIn struct {
	AppName   string
	VersionID string
}

type ActivateVersionOut struct {
	PreviousVersionID string // for undo
}

func ActivateVersion(ctx context.Context, in ActivateVersionIn) (ActivateVersionOut, error) {
	client := saga.Get[*apiserver.Client](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	// Get current app state to capture previous version
	var app core_v1alpha.App
	if err := client.Get(ctx, in.AppName, &app); err != nil {
		return ActivateVersionOut{}, fmt.Errorf("app not found: %w", err)
	}

	previousVersion := string(app.ActiveVersion)

	// Update to new version
	app.ActiveVersion = entity.Id(in.VersionID)
	if err := client.Update(ctx, &app); err != nil {
		return ActivateVersionOut{}, fmt.Errorf("failed to activate version: %w", err)
	}

	buildLog.record(fmt.Sprintf("Activated version %s (previous: %s)", in.VersionID, previousVersion))
	return ActivateVersionOut{PreviousVersionID: previousVersion}, nil
}

func UndoActivateVersion(ctx context.Context, in ActivateVersionIn, out ActivateVersionOut) error {
	client := saga.Get[*apiserver.Client](ctx)
	buildLog := saga.Get[*BuildLog](ctx)

	var app core_v1alpha.App
	if err := client.Get(ctx, in.AppName, &app); err != nil {
		return fmt.Errorf("app not found: %w", err)
	}

	// Restore previous version (may be empty for first deploy)
	app.ActiveVersion = entity.Id(out.PreviousVersionID)
	if err := client.Update(ctx, &app); err != nil {
		return fmt.Errorf("failed to restore version: %w", err)
	}

	buildLog.record(fmt.Sprintf("Restored previous version %s", out.PreviousVersionID))
	return nil
}

// --- Tests ---

func TestBuildSaga_Success(t *testing.T) {
	ctx := context.Background()

	// Set up in-memory entity server
	inmem, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	// Create an app to deploy to
	app := &core_v1alpha.App{}
	_, err := inmem.Client.Create(ctx, "myapp", app)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Set up stubbed collaborators
	stager := NewSourceStager()
	buildKit := &BuildKit{
		NextDigest:   "sha256:abc123def456",
		NextImageURL: "registry.example.com/myapp:v1",
	}
	buildLog := &BuildLog{}

	// Simulate: caller registers stream BEFORE starting saga
	// The stream itself is non-serializable, but the ID is just a string
	fakeStream := &fakeReader{data: []byte("fake tar data")}
	stager.RegisterStream("stream-123", fakeStream)

	// Define and register the saga
	registry := saga.NewRegistry()
	saga.Define("deploy-app").
		Using(inmem.Client).
		Using(stager).
		Using(buildKit).
		Using(buildLog).
		Action(StageSource).Undo(UndoStageSource).
		Action(BuildImage).Undo(UndoBuildImage).
		Action(ResolveArtifact).Undo(UndoResolveArtifact).
		Action(CreateVersion).Undo(UndoCreateVersion).
		Action(ActivateVersion).Undo(UndoActivateVersion).
		RegisterTo(registry)

	storage := saga.NewMemoryStorage()
	executor := saga.NewExecutor(storage, saga.WithRegistry(registry))

	// Execute the saga - note we pass streamid, not the stream itself!
	err = executor.Start("deploy-app").
		Input("appname", "myapp").
		Input("streamid", "stream-123").
		WithID("deploy-1").
		Execute(ctx)

	if err != nil {
		t.Fatalf("saga failed: %v", err)
	}

	// Verify the entities were created correctly
	var finalApp core_v1alpha.App
	if err := inmem.Client.Get(ctx, "myapp", &finalApp); err != nil {
		t.Fatalf("failed to get app: %v", err)
	}

	if finalApp.ActiveVersion == "" {
		t.Error("app should have an active version")
	}

	// Verify artifact was created
	var artifact core_v1alpha.Artifact
	err = inmem.Client.OneAtIndex(ctx,
		entity.String(core_v1alpha.ArtifactManifestDigestId, "sha256:abc123def456"),
		&artifact)
	if err != nil {
		t.Fatalf("artifact should exist: %v", err)
	}

	// Verify BuildKit received the staged path
	if buildKit.BuiltFromDir != "/tmp/staged/stream-123" {
		t.Errorf("expected build from /tmp/staged/stream-123, got %s", buildKit.BuiltFromDir)
	}

	// Check the build log
	t.Log("Build log:")
	for _, event := range buildLog.events {
		t.Logf("  - %s", event)
	}

	if len(buildLog.events) != 5 {
		t.Errorf("expected 5 events, got %d", len(buildLog.events))
	}
}

// fakeReader is a simple io.Reader for testing
type fakeReader struct {
	data []byte
	pos  int
}

func (r *fakeReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestBuildSaga_BuildFailure_CompensatesStaging(t *testing.T) {
	ctx := context.Background()

	// Set up in-memory entity server
	inmem, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	// Create an app
	app := &core_v1alpha.App{}
	_, err := inmem.Client.Create(ctx, "myapp", app)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Set up collaborators
	stager := NewSourceStager()
	stager.RegisterStream("stream-456", &fakeReader{data: []byte("tar data")})

	// BuildKit will fail
	buildKit := &BuildKit{
		ShouldFail:  true,
		FailMessage: "build timeout",
	}
	buildLog := &BuildLog{}

	registry := saga.NewRegistry()
	saga.Define("deploy-app-fail-build").
		Using(inmem.Client).
		Using(stager).
		Using(buildKit).
		Using(buildLog).
		Action(StageSource).Undo(UndoStageSource).
		Action(BuildImage).Undo(UndoBuildImage).
		Action(ResolveArtifact).Undo(UndoResolveArtifact).
		Action(CreateVersion).Undo(UndoCreateVersion).
		Action(ActivateVersion).Undo(UndoActivateVersion).
		RegisterTo(registry)

	storage := saga.NewMemoryStorage()
	executor := saga.NewExecutor(storage, saga.WithRegistry(registry))

	err = executor.Start("deploy-app-fail-build").
		Input("appname", "myapp").
		Input("streamid", "stream-456").
		Execute(ctx)

	if err == nil {
		t.Fatal("saga should have failed")
	}

	t.Log("Build log:")
	for _, event := range buildLog.events {
		t.Logf("  - %s", event)
	}

	// Should have: stage success, build failure, then undo stage
	// StageSource succeeded, BuildImage failed, UndoStageSource runs
	if len(buildLog.events) != 3 {
		t.Errorf("expected 3 events (stage, build fail, undo stage), got %d", len(buildLog.events))
	}

	// Verify staged source was cleaned up
	if len(stager.staged) != 0 {
		t.Error("staged source should have been cleaned up")
	}

	// App should have no active version
	var finalApp core_v1alpha.App
	if err := inmem.Client.Get(ctx, "myapp", &finalApp); err != nil {
		t.Fatalf("failed to get app: %v", err)
	}
	if finalApp.ActiveVersion != "" {
		t.Error("app should not have an active version after failed build")
	}
}

func TestBuildSaga_ActivationFailure_Compensates(t *testing.T) {
	ctx := context.Background()

	// Set up in-memory entity server
	inmem, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	// Create app
	app := &core_v1alpha.App{}
	appID, err := inmem.Client.Create(ctx, "myapp", app)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	stager := NewSourceStager()
	stager.RegisterStream("stream-789", &fakeReader{data: []byte("tar data")})

	buildKit := &BuildKit{
		NextDigest:   "sha256:abc123def456",
		NextImageURL: "registry.example.com/myapp:v1",
	}
	buildLog := &BuildLog{}

	// We'll use a custom ActivateVersion that fails
	failingActivate := func(ctx context.Context, in ActivateVersionIn) (ActivateVersionOut, error) {
		buildLog.record("Activation failed: simulated error")
		return ActivateVersionOut{}, errors.New("simulated activation failure")
	}

	registry := saga.NewRegistry()
	saga.Define("deploy-app-fail-late").
		Using(inmem.Client).
		Using(stager).
		Using(buildKit).
		Using(buildLog).
		Action(StageSource).Undo(UndoStageSource).
		Action(BuildImage).Undo(UndoBuildImage).
		Action(ResolveArtifact).Undo(UndoResolveArtifact).
		Action(CreateVersion).Undo(UndoCreateVersion).
		Action("failing-activate", failingActivate).Undo(UndoActivateVersion).
		RegisterTo(registry)

	storage := saga.NewMemoryStorage()
	executor := saga.NewExecutor(storage, saga.WithRegistry(registry))

	err = executor.Start("deploy-app-fail-late").
		Input("appname", "myapp").
		Input("streamid", "stream-789").
		Execute(ctx)

	if err == nil {
		t.Fatal("saga should have failed")
	}

	t.Log("Build log:")
	for _, event := range buildLog.events {
		t.Logf("  - %s", event)
	}

	// Should have: stage, build, artifact, version, activation fail,
	// then compensations: version deleted, artifact deleted, build cleanup, stage cleanup
	if len(buildLog.events) < 8 {
		t.Errorf("expected at least 8 events, got %d", len(buildLog.events))
	}

	// Verify cleanup: no artifact should exist
	var artifact core_v1alpha.Artifact
	err = inmem.Client.OneAtIndex(ctx,
		entity.String(core_v1alpha.ArtifactManifestDigestId, "sha256:abc123def456"),
		&artifact)
	if err == nil {
		t.Error("artifact should have been deleted during compensation")
	}

	// Verify staged source was cleaned up
	if len(stager.staged) != 0 {
		t.Error("staged source should have been cleaned up")
	}

	// App should still exist but have no active version
	var finalApp core_v1alpha.App
	if err := inmem.Client.GetById(ctx, appID, &finalApp); err != nil {
		t.Fatalf("app should still exist: %v", err)
	}
	if finalApp.ActiveVersion != "" {
		t.Error("app should not have an active version after rollback")
	}
}

func TestBuildSaga_ExistingArtifact_Reused(t *testing.T) {
	ctx := context.Background()

	// Set up in-memory entity server
	inmem, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	// Create app
	app := &core_v1alpha.App{}
	appID, err := inmem.Client.Create(ctx, "myapp", app)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Pre-create an artifact with the same digest (simulating a rebuild)
	existingArtifact := &core_v1alpha.Artifact{
		App:            appID,
		ManifestDigest: "sha256:abc123def456",
		Status:         core_v1alpha.ACTIVE,
	}
	existingArtifactID, err := inmem.Client.Create(ctx, "myapp-existing", existingArtifact)
	if err != nil {
		t.Fatalf("failed to create existing artifact: %v", err)
	}

	stager := NewSourceStager()
	stager.RegisterStream("stream-999", &fakeReader{data: []byte("tar data")})

	buildKit := &BuildKit{
		NextDigest:   "sha256:abc123def456", // Same digest
		NextImageURL: "registry.example.com/myapp:v1",
	}
	buildLog := &BuildLog{}

	registry := saga.NewRegistry()
	saga.Define("deploy-app-reuse").
		Using(inmem.Client).
		Using(stager).
		Using(buildKit).
		Using(buildLog).
		Action(StageSource).Undo(UndoStageSource).
		Action(BuildImage).Undo(UndoBuildImage).
		Action(ResolveArtifact).Undo(UndoResolveArtifact).
		Action(CreateVersion).Undo(UndoCreateVersion).
		Action(ActivateVersion).Undo(UndoActivateVersion).
		RegisterTo(registry)

	storage := saga.NewMemoryStorage()
	executor := saga.NewExecutor(storage, saga.WithRegistry(registry))

	err = executor.Start("deploy-app-reuse").
		Input("appname", "myapp").
		Input("streamid", "stream-999").
		Execute(ctx)

	if err != nil {
		t.Fatalf("saga failed: %v", err)
	}

	t.Log("Build log:")
	for _, event := range buildLog.events {
		t.Logf("  - %s", event)
	}

	// Verify the existing artifact was reused
	found := false
	for _, event := range buildLog.events {
		if event == fmt.Sprintf("Found existing artifact %s for digest sha256:abc123def456", existingArtifactID) {
			found = true
			break
		}
	}
	if !found {
		t.Error("should have found and reused existing artifact")
	}
}

// TestBuildSaga_StreamLost_FailsGracefully demonstrates what happens when we try
// to recover a saga but the stream was lost (simulating a crash before staging completed)
func TestBuildSaga_StreamLost_FailsGracefully(t *testing.T) {
	ctx := context.Background()

	inmem, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	app := &core_v1alpha.App{}
	_, err := inmem.Client.Create(ctx, "myapp", app)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Stager with NO stream registered - simulates crash where stream was lost
	stager := NewSourceStager()
	// Note: NOT calling stager.RegisterStream()

	buildKit := &BuildKit{
		NextDigest:   "sha256:abc123def456",
		NextImageURL: "registry.example.com/myapp:v1",
	}
	buildLog := &BuildLog{}

	registry := saga.NewRegistry()
	saga.Define("deploy-app-lost-stream").
		Using(inmem.Client).
		Using(stager).
		Using(buildKit).
		Using(buildLog).
		Action(StageSource).Undo(UndoStageSource).
		Action(BuildImage).Undo(UndoBuildImage).
		Action(ResolveArtifact).Undo(UndoResolveArtifact).
		Action(CreateVersion).Undo(UndoCreateVersion).
		Action(ActivateVersion).Undo(UndoActivateVersion).
		RegisterTo(registry)

	storage := saga.NewMemoryStorage()
	executor := saga.NewExecutor(storage, saga.WithRegistry(registry))

	err = executor.Start("deploy-app-lost-stream").
		Input("appname", "myapp").
		Input("streamid", "stream-gone").
		Execute(ctx)

	if err == nil {
		t.Fatal("saga should have failed - stream was lost")
	}

	t.Logf("Expected failure: %v", err)

	// The saga should fail at StageSource with "stream not found"
	// Since StageSource is the first action, there's nothing to compensate
	t.Log("Build log:")
	for _, event := range buildLog.events {
		t.Logf("  - %s", event)
	}

	// Only the failure should be logged
	if len(buildLog.events) != 1 {
		t.Errorf("expected 1 event (stage failure), got %d", len(buildLog.events))
	}
}
