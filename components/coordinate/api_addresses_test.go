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

	publicIPv4 := net.ParseIP("203.0.113.10")
	publicIPv6 := net.ParseIP("2001:db8::10")
	privateIP := net.ParseIP("10.0.0.5")

	tests := []struct {
		name           string
		additionalIPs  []net.IP
		discoveredIPs  []net.IP
		netcheckResult *cloudauth.NetcheckDualStackResult
		wantContains   []string
		wantExcludes   []string
	}{
		{
			name:          "no netcheck with discovered public IPs",
			discoveredIPs: []net.IP{publicIPv4, privateIP},
			wantContains:  append(localhost, "203.0.113.10:8443", "10.0.0.5:8443"),
		},
		{
			// MIR-1018: when netcheck ran with a valid public source IP
			// but every probed port failed, we now trust the negative
			// result and drop global-unicast discovered IPs in that
			// family. Private LAN IPs are still kept.
			name:          "netcheck proved IPv4 unreachable drops public discovered IP",
			discoveredIPs: []net.IP{publicIPv4, privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "203.0.113.10",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "tcp", Reachable: false},
					},
				},
			},
			wantContains: append(localhost, "10.0.0.5:8443"),
			wantExcludes: []string{"203.0.113.10:8443"},
		},
		{
			// Netcheck failing entirely (e.g. no cloud connectivity) still
			// lets discovered public IPs pass through as a fallback.
			name:           "netcheck not run keeps discovered public IPs",
			discoveredIPs:  []net.IP{publicIPv4, privateIP},
			netcheckResult: nil,
			wantContains:   append(localhost, "203.0.113.10:8443", "10.0.0.5:8443"),
		},
		{
			// MIR-1018: CGNAT addresses (tailscale tailnet / ISP CGNAT)
			// are filtered out of the discovered list by default.
			name:          "CGNAT discovered IP is filtered",
			discoveredIPs: []net.IP{net.ParseIP("100.107.209.9"), privateIP},
			wantContains:  append(localhost, "10.0.0.5:8443"),
			wantExcludes:  []string{"100.107.209.9:8443"},
		},
		{
			// Users who explicitly want a CGNAT address advertised can
			// still set it as an AdditionalIP.
			name:          "CGNAT AdditionalIP is kept",
			additionalIPs: []net.IP{net.ParseIP("100.107.209.9")},
			wantContains:  append(localhost, "100.107.209.9:8443"),
		},
		{
			name:          "netcheck ran and found reachable addresses",
			discoveredIPs: []net.IP{publicIPv4, privateIP},
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
			name:         "no IPs and no netcheck",
			wantContains: localhost,
		},
		{
			name:          "netcheck replaces discovered public IP with different source",
			discoveredIPs: []net.IP{publicIPv4, privateIP},
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
			discoveredIPs: []net.IP{publicIPv4, privateIP},
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
			discoveredIPs: []net.IP{publicIPv4, privateIP},
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
		{
			name:          "user-provided AdditionalIPs always included even with netcheck",
			additionalIPs: []net.IP{publicIPv4},
			discoveredIPs: []net.IP{privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "198.51.100.1",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "tcp", Reachable: true},
					},
				},
			},
			// User-provided public IP is always kept, netcheck address is also added
			wantContains: append(localhost, "203.0.113.10:8443", "198.51.100.1:8443", "10.0.0.5:8443"),
		},
		{
			name:          "mixed-family: IPv4 reachable, discovered IPv6 preserved",
			discoveredIPs: []net.IP{publicIPv4, publicIPv6, privateIP},
			netcheckResult: &cloudauth.NetcheckDualStackResult{
				IPv4: &cloudauth.NetcheckResponse{
					SourceAddress: "203.0.113.10",
					Results: []cloudauth.NetcheckResult{
						{Port: 8443, Protocol: "https", Reachable: true},
					},
				},
				IPv6: nil,
			},
			// IPv4 discovered public IP is replaced by the netcheck-confirmed
			// one. IPv6 netcheck didn't run, so we don't know whether the
			// discovered IPv6 is reachable — keep it as a fallback on a
			// per-family basis, same as we would if IPv4 netcheck had failed.
			wantContains: append(localhost, "203.0.113.10:8443", "[2001:db8::10]:8443", "10.0.0.5:8443"),
			wantExcludes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Coordinator{
				CoordinatorConfig: CoordinatorConfig{
					Address:       listenAddr,
					AdditionalIPs: tt.additionalIPs,
					DiscoveredIPs: tt.discoveredIPs,
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
