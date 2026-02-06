package runner

import (
	"testing"

	"miren.dev/runtime/pkg/joincode"
)

func TestJoinCodeIntegration(t *testing.T) {
	code, err := joincode.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !joincode.Validate(code) {
		t.Errorf("Generated code %q did not validate", code)
	}

	hash := joincode.Hash(code)
	if hash == "" {
		t.Error("Hash() returned empty string")
	}

	if len(hash) != 64 {
		t.Errorf("Hash() returned string of length %d, expected 64", len(hash))
	}
}
