package commands

import (
	"fmt"

	"miren.dev/runtime/api/ingress"
	"miren.dev/runtime/api/ingress/ingress_v1alpha"
)

func RouteUnwaf(ctx *Context, opts struct {
	Host    string `position:"0" usage:"Hostname for the route (e.g., example.com); omit and pass --default for the default route"`
	Default bool   `long:"default" description:"Disable WAF on the default route (instead of a hostname)"`
	ConfigCentric
}) error {
	if opts.Host == "" && !opts.Default {
		return fmt.Errorf("either a hostname or --default must be specified")
	}

	if opts.Host != "" && opts.Default {
		return fmt.Errorf("--default cannot be used with a hostname")
	}

	client, err := ctx.RPCClient("entities")
	if err != nil {
		return err
	}

	ic := ingress.NewClient(ctx.Log, client)

	var route *ingress_v1alpha.HttpRoute
	var routeLabel string

	if opts.Default {
		route, err = ic.LookupDefault(ctx)
		if err != nil {
			return fmt.Errorf("failed to lookup default route: %w", err)
		}
		if route == nil {
			return fmt.Errorf("no default route configured")
		}
		routeLabel = "default"
	} else {
		route, err = ic.Lookup(ctx, opts.Host)
		if err != nil {
			return fmt.Errorf("failed to lookup route: %w", err)
		}
		if route == nil {
			return fmt.Errorf("route not found for host: %s", opts.Host)
		}
		routeLabel = opts.Host
	}

	if route.WafLevel == 0 {
		ctx.Printf("WAF is not enabled on route: %s\n", routeLabel)
		return nil
	}

	_, err = ic.SetRouteWAFLevelOnRoute(ctx, route, 0)
	if err != nil {
		return fmt.Errorf("failed to disable WAF on route: %w", err)
	}

	ctx.Printf("WAF disabled on route: %s\n", routeLabel)
	return nil
}
