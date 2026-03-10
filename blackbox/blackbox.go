//go:build blackbox

// Package blackbox contains black-box tests that exercise the miren CLI as a
// real subprocess against a running cluster. Tests in this package use no
// internal Go imports from the runtime — they interact exclusively through the
// CLI binary and HTTP.
//
// Run with: go test -tags blackbox -timeout 10m -v -count=1 ./blackbox/...
//
// Prerequisites: a running dev environment (make dev).
package blackbox
