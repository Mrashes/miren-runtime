package activator

import "miren.dev/runtime/api/compute/compute_v1alpha"

// NewTestLease creates a Lease with the given sandbox, size, and URL.
// This is only intended for use in tests outside the activator package.
func NewTestLease(sb *compute_v1alpha.Sandbox, size int, url string) *Lease {
	return &Lease{
		sandbox: sb,
		Size:    size,
		URL:     url,
	}
}
