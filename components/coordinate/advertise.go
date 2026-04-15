package coordinate

import (
	"net"
	"strconv"

	"miren.dev/runtime/pkg/cloudauth"
)

// AdvertiseInput is the raw input for computing the set of API addresses
// the server should advertise to clients and to miren.cloud.
type AdvertiseInput struct {
	// ListenAddr is the server's own listen address (e.g. "0.0.0.0:8443").
	// Included in the advertised list only if it has a literal IP host.
	ListenAddr string

	// AdditionalIPs are user-configured IPs that must always be advertised.
	AdditionalIPs []net.IP

	// DiscoveredIPs are interface-scanned IPs from the local host.
	DiscoveredIPs []net.IP

	// Netcheck is the result of the dual-stack netcheck, if one has run.
	// A nil pointer means netcheck never ran / failed entirely.
	Netcheck *cloudauth.NetcheckDualStackResult

	// Port is the port to append to bare IPs (defaults to 8443).
	Port int
}

// AdvertiseCandidate describes one candidate address the advertise logic
// considered, and whether it ended up in the final advertised set. Used by
// both production (building the final list) and debug tooling (explaining
// the decision for every IP).
type AdvertiseCandidate struct {
	Source         string // "listen", "localhost", "additional", "discovered", "netcheck"
	HostPort       string
	IP             net.IP
	Classification string // loopback / link-local / private / global-unicast / other
	Included       bool
	Reason         string
}

// ComputeAdvertise is the single source of truth for computing the addresses
// the server advertises. It returns the ordered list of candidates (including
// rejected ones, so callers can explain why) and the final list of advertised
// host:port strings.
//
// Filtering rules:
//
//  1. Listen address: included if it parses as host:port with a literal IP.
//  2. Localhost (127.0.0.1, ::1): always included.
//  3. AdditionalIPs: always included (user-curated).
//  4. DiscoveredIPs:
//     a. CGNAT addresses (100.64.0.0/10) are dropped. This range is used
//     by tailscale tailnets and carrier-grade NAT; advertising them to
//     a generic client is misleading. Users who want a CGNAT address
//     advertised can pass it via AdditionalIPs.
//     b. Other private / loopback / link-local IPs are always included —
//     they may serve clients on the same LAN and we can't tell from
//     here which private ranges are reachable.
//     c. Public (global-unicast, non-private) IPs are dropped if netcheck
//     ran for that address family and proved the family unreachable
//     (valid public source IP, zero reachable ports), or if netcheck
//     found reachable addresses (in which case the confirmed netcheck
//     source is advertised instead).
//     d. Otherwise (netcheck didn't run or source was invalid) they are
//     kept as a fallback.
//  5. Netcheck public addresses: included when reachable on at least one port.
func ComputeAdvertise(in AdvertiseInput) ([]AdvertiseCandidate, []string) {
	port := in.Port
	if port == 0 {
		port = 8443
	}
	portStr := strconv.Itoa(port)

	var cands []AdvertiseCandidate
	var final []string
	seen := make(map[string]struct{})

	add := func(c AdvertiseCandidate) {
		cands = append(cands, c)
		if !c.Included {
			return
		}
		if _, ok := seen[c.HostPort]; ok {
			return
		}
		seen[c.HostPort] = struct{}{}
		final = append(final, c.HostPort)
	}

	// 1. Listen address.
	if in.ListenAddr != "" {
		host, _, err := net.SplitHostPort(in.ListenAddr)
		if err == nil && net.ParseIP(host) != nil {
			ip := net.ParseIP(host)
			add(AdvertiseCandidate{
				Source:         "listen",
				HostPort:       in.ListenAddr,
				IP:             ip,
				Classification: classify(ip),
				Included:       true,
				Reason:         "server listen address",
			})
		} else {
			add(AdvertiseCandidate{
				Source:   "listen",
				HostPort: in.ListenAddr,
				Included: false,
				Reason:   "not a literal IP host",
			})
		}
	}

	// 2. Localhost.
	for _, lh := range []string{
		net.JoinHostPort("127.0.0.1", portStr),
		net.JoinHostPort("::1", portStr),
	} {
		host, _, _ := net.SplitHostPort(lh)
		add(AdvertiseCandidate{
			Source:         "localhost",
			HostPort:       lh,
			IP:             net.ParseIP(host),
			Classification: "loopback",
			Included:       true,
			Reason:         "always advertised",
		})
	}

	// 3. AdditionalIPs.
	for _, ip := range in.AdditionalIPs {
		if ip == nil {
			continue
		}
		add(AdvertiseCandidate{
			Source:         "additional",
			HostPort:       net.JoinHostPort(ip.String(), portStr),
			IP:             ip,
			Classification: classify(ip),
			Included:       true,
			Reason:         "user-configured",
		})
	}

	// Compute per-family netcheck state.
	v4State := netcheckFamilyState(familyIPv4, in.Netcheck)
	v6State := netcheckFamilyState(familyIPv6, in.Netcheck)

	// 4. DiscoveredIPs.
	for _, ip := range in.DiscoveredIPs {
		if ip == nil {
			continue
		}
		hp := net.JoinHostPort(ip.String(), portStr)
		cand := AdvertiseCandidate{
			Source:         "discovered",
			HostPort:       hp,
			IP:             ip,
			Classification: classify(ip),
		}

		if isCGNAT(ip) {
			cand.Included = false
			cand.Reason = "CGNAT 100.64.0.0/10 (e.g. tailscale)"
			add(cand)
			continue
		}

		isPublicCandidate := !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast()
		if !isPublicCandidate {
			cand.Included = true
			cand.Reason = "private/loopback/link-local, kept for LAN clients"
			add(cand)
			continue
		}

		state := v4State
		if ip.To4() == nil {
			state = v6State
		}
		switch state {
		case netcheckReachable:
			cand.Included = false
			cand.Reason = "replaced by netcheck-confirmed public address"
		case netcheckUnreachable:
			cand.Included = false
			cand.Reason = "address family proven unreachable by netcheck"
		default:
			cand.Included = true
			cand.Reason = "no netcheck override"
		}
		add(cand)
	}

	// 5. Netcheck public addresses.
	for _, hp := range publicAddressesFromNetcheck(in.Netcheck) {
		host, _, _ := net.SplitHostPort(hp)
		ip := net.ParseIP(host)
		add(AdvertiseCandidate{
			Source:         "netcheck",
			HostPort:       hp,
			IP:             ip,
			Classification: classify(ip),
			Included:       true,
			Reason:         "netcheck confirmed reachable",
		})
	}

	return cands, final
}

type netcheckFamily int

const (
	familyIPv4 netcheckFamily = iota
	familyIPv6
)

type netcheckStatus int

const (
	netcheckNotRun netcheckStatus = iota
	netcheckUnreachable
	netcheckReachable
)

// netcheckFamilyState returns what we know about reachability for one address
// family. A nil NetcheckDualStackResult or a nil family response means "not
// run". A response with a non-public/invalid source address is also treated
// as not run (same rule runNetcheck applies). A response with a valid source
// but zero reachable ports is "proven unreachable".
func netcheckFamilyState(fam netcheckFamily, result *cloudauth.NetcheckDualStackResult) netcheckStatus {
	if result == nil {
		return netcheckNotRun
	}
	var resp *cloudauth.NetcheckResponse
	switch fam {
	case familyIPv4:
		resp = result.IPv4
	case familyIPv6:
		resp = result.IPv6
	}
	if resp == nil {
		return netcheckNotRun
	}
	src := net.ParseIP(resp.SourceAddress)
	if src == nil || !src.IsGlobalUnicast() || src.IsPrivate() {
		return netcheckNotRun
	}
	for _, r := range resp.Results {
		if r.Reachable {
			return netcheckReachable
		}
	}
	return netcheckUnreachable
}

// publicAddressesFromNetcheck returns netcheck-confirmed reachable host:port
// strings. Mirrors the old (*Coordinator).publicAddresses() but as a pure
// function so it can be shared with debug tooling.
func publicAddressesFromNetcheck(result *cloudauth.NetcheckDualStackResult) []string {
	if result == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var addrs []string
	for _, resp := range []*cloudauth.NetcheckResponse{result.IPv4, result.IPv6} {
		if resp == nil || resp.SourceAddress == "" {
			continue
		}
		src := net.ParseIP(resp.SourceAddress)
		if src == nil || !src.IsGlobalUnicast() || src.IsPrivate() {
			continue
		}
		for _, r := range resp.Results {
			if !r.Reachable {
				continue
			}
			hp := net.JoinHostPort(resp.SourceAddress, strconv.Itoa(r.Port))
			if _, ok := seen[hp]; ok {
				continue
			}
			seen[hp] = struct{}{}
			addrs = append(addrs, hp)
		}
	}
	return addrs
}

// isCGNAT reports whether ip falls in the 100.64.0.0/10 Carrier-Grade NAT
// range (RFC 6598). Tailscale tailnet addresses also live in this range,
// so filtering CGNAT out of discovered-IP lists keeps them from being
// advertised to clients who aren't on the tailnet.
func isCGNAT(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 100 && ip4[1]&0xc0 == 0x40
}

// classify returns a short string describing the kind of address, for
// diagnostic output.
func classify(ip net.IP) string {
	if ip == nil {
		return "unknown"
	}
	switch {
	case ip.IsLoopback():
		return "loopback"
	case ip.IsLinkLocalUnicast():
		return "link-local"
	case ip.IsPrivate():
		return "private"
	case ip.IsGlobalUnicast():
		return "global-unicast"
	default:
		return "other"
	}
}
