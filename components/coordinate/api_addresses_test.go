package coordinate

import (
	"log/slog"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"miren.dev/runtime/pkg/cloudauth"
)

func TestApiAddresses(t *testing.T) {
	const listenAddr = "0.0.0.0:8443"

	localhost := []string{"0.0.0.0:8443", "127.0.0.1:8443", "[::1]:8443"}

	publicIP := net.ParseIP("203.0.113.10")
	privateIP := net.ParseIP("10.0.0.5")

	tests := []struct {
		name           string
		additionalIPs  []net.IP
		netcheckResult *cloudauth.NetcheckDualStackResult
		wantContains   []string
		wantExcludes   []string
	}{
		{
			name:          "no netcheck with public AdditionalIPs",
			additionalIPs: []net.IP{publicIP, privateIP},
			wantContains:  append(localhost, "203.0.113.10:8443", "10.0.0.5:8443"),
		},
		{
			name:          "netcheck ran but found nothing reachable",
			additionalIPs: []net.IP{publicIP, privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "203.0.113.10",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "tcp", Reachable: false},
					},
				},
			},
			// When netcheck finds nothing reachable, keep discovered public IPs
			// as a fallback rather than dropping them.
			wantContains: append(localhost, "203.0.113.10:8443", "10.0.0.5:8443"),
		},
		{
			name:          "netcheck ran and found reachable addresses",
			additionalIPs: []net.IP{publicIP, privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "203.0.113.10",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "tcp", Reachable: true},
					},
				},
			},
			wantContains: append(localhost, "10.0.0.5:8443", "203.0.113.10:8443"),
		},
		{
			name:         "no AdditionalIPs and no netcheck",
			wantContains: localhost,
		},
		{
			name:          "netcheck replaces public AdditionalIP with different source",
			additionalIPs: []net.IP{publicIP, privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "198.51.100.1",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "tcp", Reachable: true},
					},
				},
			},
			wantContains: append(localhost, "198.51.100.1:8443", "10.0.0.5:8443"),
			wantExcludes: []string{"203.0.113.10:8443"},
		},
		{
			name:          "dual-stack netcheck with both families reachable",
			additionalIPs: []net.IP{publicIP, privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "203.0.113.10",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "https", Reachable: true},
					},
				},
				IPv6: &cloudauth.NetcheckResponse{
					SourceAddress: "2001:db8::1",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "https", Reachable: true},
					},
				},
			},
			wantContains: append(localhost, "203.0.113.10:8443", "[2001:db8::1]:8443", "10.0.0.5:8443"),
		},
		{
			name:          "dual-stack netcheck with only IPv4 reachable",
			additionalIPs: []net.IP{publicIP, privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "203.0.113.10",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "https", Reachable: true},
					},
				},
				IPv6: nil,
			},
			wantContains: append(localhost, "203.0.113.10:8443", "10.0.0.5:8443"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Coordinator{
				CoordinatorConfig: CoordinatorConfig{
					Address:       listenAddr,
					AdditionalIPs: tt.additionalIPs,
				},
				Log:            slog.Default(),
				netcheckResult: tt.netcheckResult,
			}

			got := c.apiAddresses()

			for _, want := range tt.wantContains {
				assert.Contains(t, got, want, "expected %q in result", want)
			}
			for _, excluded := range tt.wantExcludes {
				assert.NotContains(t, got, excluded, "expected %q to be excluded", excluded)
			}
		})
	}
}
