package commands

import (
	"fmt"

	"miren.dev/runtime/api/ingress"
)

func RouteOidcDisable(ctx *Context, opts struct {
	Host string `position:"0" usage:"Hostname for the route (e.g., example.com)" required:"true"`
	ConfigCentric
}) error {
	client, err := ctx.RPCClient("entities")
	if err != nil {
		return err
	}

	ic := ingress.NewClient(ctx.Log, client)

	// Look up existing route
	route, err := ic.Lookup(ctx, opts.Host)
	if err != nil {
		return fmt.Errorf("failed to lookup route: %w", err)
	}

	if route == nil {
		return fmt.Errorf("route not found for host: %s", opts.Host)
	}

	// Check if OIDC is configured
	if route.OidcConfig.Empty() {
		ctx.Printf("OIDC is not configured for route: %s\n", opts.Host)
		return nil
	}

	// Clear OIDC config
	_, err = ic.ClearOIDCConfig(ctx, opts.Host)
	if err != nil {
		return fmt.Errorf("failed to disable OIDC: %w", err)
	}

	ctx.Printf("OIDC disabled for route: %s\n", opts.Host)
	return nil
}
