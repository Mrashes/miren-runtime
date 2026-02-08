package commands

import (
	"fmt"

	"miren.dev/runtime/api/ingress"
)

func RouteOidcShow(ctx *Context, opts struct {
	Host string `position:"0" usage:"Hostname for the route (e.g., example.com)" required:"true"`
	FormatOptions
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
		if opts.IsJSON() {
			return PrintJSON(map[string]interface{}{
				"host":         opts.Host,
				"oidc_enabled": false,
				"oidc_config":  nil,
			})
		}
		ctx.Printf("OIDC is not configured for route: %s\n", opts.Host)
		return nil
	}

	// Display OIDC config
	if opts.IsJSON() {
		type OIDCConfigJSON struct {
			Host          string              `json:"host"`
			OIDCEnabled   bool                `json:"oidc_enabled"`
			ProviderURL   string              `json:"provider_url"`
			ClientID      string              `json:"client_id"`
			Scopes        string              `json:"scopes"`
			ClaimMappings []map[string]string `json:"claim_mappings,omitempty"`
		}

		var mappings []map[string]string
		for _, m := range route.OidcConfig.ClaimMappings {
			mappings = append(mappings, map[string]string{
				"claim":  m.Claim,
				"header": m.Header,
			})
		}

		return PrintJSON(OIDCConfigJSON{
			Host:          opts.Host,
			OIDCEnabled:   true,
			ProviderURL:   route.OidcConfig.ProviderUrl,
			ClientID:      route.OidcConfig.ClientId,
			Scopes:        route.OidcConfig.Scopes,
			ClaimMappings: mappings,
		})
	}

	ctx.Printf("OIDC Configuration for route: %s\n\n", opts.Host)
	ctx.Printf("Enabled:      Yes\n")
	ctx.Printf("Provider URL: %s\n", route.OidcConfig.ProviderUrl)
	ctx.Printf("Client ID:    %s\n", route.OidcConfig.ClientId)
	ctx.Printf("Scopes:       %s\n", route.OidcConfig.Scopes)

	if len(route.OidcConfig.ClaimMappings) > 0 {
		ctx.Printf("\nClaim Mappings:\n")
		for _, mapping := range route.OidcConfig.ClaimMappings {
			ctx.Printf("  %s → %s\n", mapping.Claim, mapping.Header)
		}
	}

	return nil
}
