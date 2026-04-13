package labs

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestDisableFeatureWithPrefix(t *testing.T) {
	Reset()

	// Enable first, then disable
	Init(nil, []string{"globalrouter", "-globalrouter"})

	if GlobalRouter() {
		t.Error("GlobalRouter should be disabled after '-globalrouter'")
	}
}

func TestCaseInsensitiveFeatureNames(t *testing.T) {
	Reset()

	Init(nil, []string{"GlobalRouter", "ADMINAPI"})

	if !GlobalRouter() {
		t.Error("GlobalRouter should be enabled (case-insensitive)")
	}
	if !AdminAPI() {
		t.Error("AdminAPI should be enabled (case-insensitive)")
	}
}

func TestUnknownFeatureLogsWarning(t *testing.T) {
	Reset()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	Init(logger, []string{"unknownfeature"})

	logOutput := buf.String()
	if !strings.Contains(logOutput, "unknown labs feature flag") {
		t.Errorf("Expected warning about unknown feature, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "unknownfeature") {
		t.Errorf("Expected warning to contain the unknown feature name, got: %s", logOutput)
	}
}

func TestEmptyAndWhitespaceFlags(t *testing.T) {
	Reset()

	Init(nil, []string{"", "  ", "globalrouter", "  ", ""})

	if !GlobalRouter() {
		t.Error("GlobalRouter should be enabled despite empty/whitespace flags")
	}
}

func TestAllKeywordEnablesAllFeatures(t *testing.T) {
	Reset()

	Init(nil, []string{"all"})

	for _, name := range AllFeatures() {
		if !IsEnabled(name) {
			t.Errorf("Feature %q should be enabled after Init with 'all'", name)
		}
	}
}

func TestAllKeywordWithExclusion(t *testing.T) {
	Reset()

	Init(nil, []string{"all", "-addons"})

	for _, name := range AllFeatures() {
		if name == FeatureAddons {
			if IsEnabled(name) {
				t.Error("Addons should be disabled after 'all,-addons'")
			}
		} else {
			if !IsEnabled(name) {
				t.Errorf("Feature %q should be enabled after 'all,-addons'", name)
			}
		}
	}
}

func TestNegativeAllDisablesAll(t *testing.T) {
	Reset()

	Init(nil, []string{"globalrouter", "adminapi", "-all"})

	for _, name := range AllFeatures() {
		if IsEnabled(name) {
			t.Errorf("Feature %q should be disabled after '-all'", name)
		}
	}
}
