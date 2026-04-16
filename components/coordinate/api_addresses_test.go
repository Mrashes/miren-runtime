package coordinate

import (
	"log/slog"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"miren.dev/runtime/pkg/cloudauth"
)

func TestApiAddresses(t *testing.T) {
	// Default listen is the wildcard 0.0.0.0:8443. This is what most
	// servers actually run with, and it must never end up in the
	// advertised list that gets shipped to miren.cloud. A client reached
	// via cloud can't connect to 0.0.0.0 or to loopback.
	const wildcardListen = "0.0.0.0:8443"

	// Addresses that must never appear in the advertised list for any
	// case below, since a remote client reached via miren.cloud can't
	// use them.
	nonRoutable := []string{
		"0.0.0.0:8443",
		"[::]:8443",
		"127.0.0.1:8443",
		"[::1]:8443",
	}

	publicIPv4 := net.ParseIP("203.0.113.10")
	publicIPv6 := net.ParseIP("2001:db8::10")
	privateIP := net.ParseIP("10.0.0.5")

	tests := []struct {
		name           string
		listenAddr     string
		additionalIPs  []net.IP
		discoveredIPs  []net.IP
		netcheckResult *cloudauth.NetcheckDualStackResult
		wantContains   []string
		wantExcludes   []string
	}{
		{
			name:          "no netcheck with discovered public IPs",
			discoveredIPs: []net.IP{publicIPv4, privateIP},
			wantContains:  []string{"203.0.113.10:8443", "10.0.0.5:8443"},
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
			wantContains: []string{"10.0.0.5:8443"},
			wantExcludes: []string{"203.0.113.10:8443"},
		},
		{
			// Netcheck failing entirely (e.g. no cloud connectivity) still
			// lets discovered public IPs pass through as a fallback.
			name:           "netcheck not run keeps discovered public IPs",
			discoveredIPs:  []net.IP{publicIPv4, privateIP},
			netcheckResult: nil,
			wantContains:   []string{"203.0.113.10:8443", "10.0.0.5:8443"},
		},
		{
			// MIR-1018: CGNAT addresses (tailscale tailnet / ISP CGNAT)
			// are filtered out of the discovered list by default.
			name:          "CGNAT discovered IP is filtered",
			discoveredIPs: []net.IP{net.ParseIP("100.107.209.9"), privateIP},
			wantContains:  []string{"10.0.0.5:8443"},
			wantExcludes:  []string{"100.107.209.9:8443"},
		},
		{
			// Users who explicitly want a CGNAT address advertised can
			// still set it as an AdditionalIP.
			name:          "CGNAT AdditionalIP is kept",
			additionalIPs: []net.IP{net.ParseIP("100.107.209.9")},
			wantContains:  []string{"100.107.209.9:8443"},
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
			wantContains: []string{"10.0.0.5:8443", "203.0.113.10:8443"},
		},
		{
			// Nothing to advertise. Should produce an empty list, not
			// a list of nonsense loopback / wildcard entries.
			name:         "no IPs and no netcheck yields empty list",
			wantContains: nil,
			wantExcludes: nonRoutable,
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
			wantContains: []string{"198.51.100.1:8443", "10.0.0.5:8443"},
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
			wantContains: []string{"203.0.113.10:8443", "[2001:db8::1]:8443", "10.0.0.5:8443"},
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
			wantContains: []string{"203.0.113.10:8443", "10.0.0.5:8443"},
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
			wantContains: []string{"203.0.113.10:8443", "198.51.100.1:8443", "10.0.0.5:8443"},
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
			wantContains: []string{"203.0.113.10:8443", "[2001:db8::10]:8443", "10.0.0.5:8443"},
		},
		{
			// A non-wildcard, non-loopback listen address IS advertised,
			// because it's a real routable bind.
			name:         "explicit non-wildcard listen address is advertised",
			listenAddr:   "198.51.100.7:8443",
			wantContains: []string{"198.51.100.7:8443"},
		},
		{
			// AdditionalIPs that are loopback/unspecified get rejected.
			name:          "loopback AdditionalIP is dropped",
			additionalIPs: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1"), net.ParseIP("0.0.0.0"), publicIPv4},
			wantContains:  []string{"203.0.113.10:8443"},
			wantExcludes:  nonRoutable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listen := tt.listenAddr
			if listen == "" {
				listen = wildcardListen
			}
			c := &Coordinator{
				CoordinatorConfig: CoordinatorConfig{
					Address:       listen,
					AdditionalIPs: tt.additionalIPs,
					DiscoveredIPs: tt.discoveredIPs,
				},
				Log:            slog.Default(),
				netcheckResult: tt.netcheckResult,
			}

			got := c.apiAddresses()

			// Non-routable addresses are NEVER advertised — assert
			// this for every case, in addition to the per-case
			// excludes.
			for _, nr := range nonRoutable {
				assert.NotContains(t, got, nr, "non-routable address %q must never be advertised", nr)
			}

			for _, want := range tt.wantContains {
				assert.Contains(t, got, want, "expected %q in result", want)
			}
			for _, excluded := range tt.wantExcludes {
				assert.NotContains(t, got, excluded, "expected %q to be excluded", excluded)
			}
		})
	}
}
