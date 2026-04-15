package commands

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"

	"miren.dev/runtime/components/coordinate"
	"miren.dev/runtime/pkg/cloudauth"
	"miren.dev/runtime/pkg/ipdiscovery"
)

// DebugAdvertise reproduces the server's public-IP advertisement logic and
// prints per-candidate classification + the final advertised list, so we can
// debug cases where the server advertises addresses that aren't actually
// reachable from clients.
//
// This mirrors the behavior of:
//   - cli/commands/server.go (ipdiscovery.DiscoverWithTimeout + link-local filter)
//   - components/coordinate/coordinate.go (runNetcheck, publicAddresses, apiAddresses)
//
// It deliberately duplicates that logic so it can run without a live
// coordinator. If the logic in either place changes, update this command too.
func DebugAdvertise(ctx *Context, opts struct {
	CloudURL      string   `long:"cloud-url" description:"Cloud URL to use for netcheck (default: https://api.miren.cloud)"`
	SkipNetcheck  bool     `long:"skip-netcheck" description:"Skip the netcheck call and only report interface scan"`
	AdditionalIPs []string `long:"additional-ip" description:"Simulate a server-configured AdditionalIP (repeatable)"`
	ListenAddr    string   `long:"listen" description:"Simulate the server's listen address (default: 0.0.0.0:8443)"`
}) error {
	cloudURL := opts.CloudURL
	if cloudURL == "" {
		cloudURL = coordinate.DefaultCloudURL
	}
	listenAddr := opts.ListenAddr
	if listenAddr == "" {
		listenAddr = "0.0.0.0:8443"
	}

	ctx.Info("debug advertise — reproducing server advertisement logic")
	ctx.Info("  cloud URL:    %s", cloudURL)
	ctx.Info("  listen:       %s", listenAddr)
	ctx.Info("  netcheck:     %s", boolWord(!opts.SkipNetcheck, "enabled", "skipped"))
	ctx.Info("")

	discoveryOpts := ipdiscovery.Options{}
	if !opts.SkipNetcheck {
		discoveryOpts.NetcheckURL = cloudURL
	}

	ctx.Info("Step 1: interface scan (+ optional netcheck IP discovery)")
	discovery, err := ipdiscovery.DiscoverWithTimeout(15*time.Second, ctx.Log, discoveryOpts)
	if err != nil {
		ctx.Warn("ipdiscovery.Discover failed: %v", err)
		return err
	}

	var discoveredIPs []net.IP
	for _, a := range discovery.Addresses {
		ip := net.ParseIP(a.IP)
		if ip == nil {
			continue
		}
		// server.go drops link-local addresses before handing the list
		// to the coordinator — mirror that here.
		if ip.IsLinkLocalUnicast() {
			ctx.Info("  %-15s %-40s [skipped: link-local]", a.Interface, a.IP)
			continue
		}
		ctx.Info("  %-15s %-40s", a.Interface, a.IP)
		discoveredIPs = append(discoveredIPs, ip)
	}
	ctx.Info("")

	var additionalIPs []net.IP
	for _, s := range opts.AdditionalIPs {
		ip := net.ParseIP(s)
		if ip == nil {
			ctx.Warn("--additional-ip %q is not a valid IP, skipping", s)
			continue
		}
		additionalIPs = append(additionalIPs, ip)
	}

	ctx.Info("Step 2: dual-stack netcheck")
	var netcheckResult *cloudauth.NetcheckDualStackResult
	if opts.SkipNetcheck {
		ctx.Info("  skipped (--skip-netcheck)")
	} else {
		ports := []cloudauth.NetcheckPort{
			{Port: 8443, Protocol: "https"},
			{Port: 8443, Protocol: "http3"},
		}
		netcheckResult, err = cloudauth.NetcheckDualStack(ctx, cloudURL, ports)
		if err != nil {
			ctx.Warn("netcheck failed: %v", err)
			netcheckResult = nil
		} else {
			printNetcheckResponse(ctx, "IPv4", netcheckResult.IPv4)
			printNetcheckResponse(ctx, "IPv6", netcheckResult.IPv6)
		}
	}
	ctx.Info("")

	// Apply the same source-address validation runNetcheck does: drop any
	// source IP that isn't a public global unicast address. A rejected
	// source family is treated as if that family's netcheck never ran.
	if netcheckResult != nil {
		if netcheckResult.IPv4 != nil {
			src := net.ParseIP(netcheckResult.IPv4.SourceAddress)
			if src == nil || !src.IsGlobalUnicast() || src.IsPrivate() {
				ctx.Info("  IPv4 source %s rejected (not public global unicast)", netcheckResult.IPv4.SourceAddress)
				netcheckResult.IPv4 = nil
			}
		}
		if netcheckResult.IPv6 != nil {
			src := net.ParseIP(netcheckResult.IPv6.SourceAddress)
			if src == nil || !src.IsGlobalUnicast() || src.IsPrivate() {
				ctx.Info("  IPv6 source %s rejected (not public global unicast)", netcheckResult.IPv6.SourceAddress)
				netcheckResult.IPv6 = nil
			}
		}
		if netcheckResult.IPv4 == nil && netcheckResult.IPv6 == nil {
			netcheckResult = nil
		}
	}

	pubAddrs := publicAddressesFromNetcheck(netcheckResult)

	ctx.Info("Step 3: per-candidate classification and inclusion decision")
	ctx.Info("  (mirrors components/coordinate/coordinate.go apiAddresses)")
	ctx.Info("")
	ctx.Info("  %-18s %-40s %-18s %s", "SOURCE", "IP:PORT", "CLASSIFICATION", "DECISION")
	ctx.Info("  %s", "------------------------------------------------------------------------------------------------")

	var finalAddrs []string

	// Listen address — only included if it has a valid IP host.
	if host, _, err := net.SplitHostPort(listenAddr); err == nil && net.ParseIP(host) != nil {
		finalAddrs = append(finalAddrs, listenAddr)
		ctx.Info("  %-18s %-40s %-18s ADVERTISED", "listen", listenAddr, classifyHostPort(listenAddr))
	} else {
		ctx.Info("  %-18s %-40s %-18s SKIPPED (not a literal IP host)", "listen", listenAddr, "-")
	}

	// Localhost — always added.
	for _, lh := range []string{"127.0.0.1:8443", "[::1]:8443"} {
		finalAddrs = append(finalAddrs, lh)
		ctx.Info("  %-18s %-40s %-18s ADVERTISED (always)", "localhost", lh, "loopback")
	}

	// Configured AdditionalIPs — always added.
	for _, ip := range additionalIPs {
		hp := net.JoinHostPort(ip.String(), "8443")
		finalAddrs = append(finalAddrs, hp)
		ctx.Info("  %-18s %-40s %-18s ADVERTISED (configured)", "additional", hp, classify(ip))
	}

	// Discovered IPs — filtered against netcheck's public addresses per
	// the current apiAddresses rule.
	for _, ip := range discoveredIPs {
		hp := net.JoinHostPort(ip.String(), "8443")
		class := classify(ip)
		isPublicCandidate := !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast()
		if len(pubAddrs) > 0 && isPublicCandidate {
			ctx.Info("  %-18s %-40s %-18s SKIPPED (replaced by netcheck public)", "discovered", hp, class)
			continue
		}
		reason := "fallback (no netcheck public addrs)"
		if !isPublicCandidate {
			reason = "private/loopback, included unconditionally"
		}
		finalAddrs = append(finalAddrs, hp)
		ctx.Info("  %-18s %-40s %-18s ADVERTISED (%s)", "discovered", hp, class, reason)
	}

	// Netcheck-reachable public addresses.
	for _, hp := range pubAddrs {
		finalAddrs = append(finalAddrs, hp)
		ctx.Info("  %-18s %-40s %-18s ADVERTISED (netcheck reachable)", "netcheck", hp, classifyHostPort(hp))
	}

	ctx.Info("")
	ctx.Info("Final advertised list (%d entries):", len(finalAddrs))
	for _, a := range finalAddrs {
		ctx.Info("  %s", a)
	}

	reachabilitySummary(ctx, netcheckResult, discoveredIPs, additionalIPs, pubAddrs)

	return nil
}

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
		if net.ParseIP(resp.SourceAddress) == nil {
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

func printNetcheckResponse(ctx *Context, family string, resp *cloudauth.NetcheckResponse) {
	if resp == nil {
		ctx.Info("  %s: no response", family)
		return
	}
	var reachable []string
	var unreachable []string
	for _, r := range resp.Results {
		entry := fmt.Sprintf("%s/%d", r.Protocol, r.Port)
		if r.Reachable {
			reachable = append(reachable, entry)
		} else {
			unreachable = append(unreachable, entry)
		}
	}
	sort.Strings(reachable)
	sort.Strings(unreachable)
	ctx.Info("  %s source=%s reachable=%v unreachable=%v duration=%dms",
		family, resp.SourceAddress, reachable, unreachable, resp.DurationMs)
}

func classify(ip net.IP) string {
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

func classifyHostPort(hp string) string {
	host, _, err := net.SplitHostPort(hp)
	if err != nil {
		return "-"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "-"
	}
	return classify(ip)
}

func boolWord(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

// reachabilitySummary highlights the specific failure modes behind MIR-1018:
// netcheck source IPs that were proven unreachable but still get advertised,
// and tailscale/bridge/ULA interfaces that sneak through the private filter.
func reachabilitySummary(ctx *Context, result *cloudauth.NetcheckDualStackResult, discovered, additional []net.IP, pubAddrs []string) {
	ctx.Info("")
	ctx.Info("Reachability notes:")

	noted := false

	// Case 1: netcheck returned a source IP but *zero* reachable ports.
	if result != nil {
		for _, entry := range []struct {
			family string
			resp   *cloudauth.NetcheckResponse
		}{
			{"IPv4", result.IPv4},
			{"IPv6", result.IPv6},
		} {
			if entry.resp == nil || entry.resp.SourceAddress == "" {
				continue
			}
			anyReachable := false
			for _, r := range entry.resp.Results {
				if r.Reachable {
					anyReachable = true
					break
				}
			}
			if !anyReachable {
				noted = true
				ctx.Warn("  netcheck %s source %s is PROVEN UNREACHABLE but still advertised via the discovered-IP fallback (MIR-1018 bug #1)",
					entry.family, entry.resp.SourceAddress)
			}
		}
	}

	// Case 2: suspicious private-range interfaces that are almost never
	// reachable from a generic client (tailscale CGNAT, docker, libvirt, ULA).
	for _, ip := range discovered {
		if reason := suspiciousPrivate(ip); reason != "" {
			noted = true
			ctx.Warn("  discovered IP %s advertised (%s) — almost certainly not reachable from an arbitrary client (MIR-1018 bug #2)",
				ip, reason)
		}
	}

	if !noted {
		ctx.Info("  none")
	}
	_ = pubAddrs
	_ = additional
}

func suspiciousPrivate(ip net.IP) string {
	if ip == nil {
		return ""
	}
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 100 && ip4[1]&0xc0 == 64:
			return "tailscale CGNAT 100.64.0.0/10"
		case ip4[0] == 172 && ip4[1] == 17:
			return "docker bridge 172.17.0.0/16"
		case ip4[0] == 192 && ip4[1] == 168 && ip4[2] == 122:
			return "libvirt default bridge 192.168.122.0/24"
		}
		return ""
	}
	// IPv6 ULA fc00::/7
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return "IPv6 ULA fc00::/7 (e.g. tailscale)"
	}
	return ""
}
