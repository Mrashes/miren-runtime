package commands

import (
	"fmt"
	"strings"

	"miren.dev/runtime/api/ingress"
	"miren.dev/runtime/api/ingress/ingress_v1alpha"
)

func RouteOidcEnable(ctx *Context, opts struct {
	Host         string   `position:"0" usage:"Hostname for the route (e.g., example.com)" required:"true"`
	ProviderURL  string   `flag:"provider-url" usage:"OIDC provider URL (e.g., https://accounts.google.com)" required:"true"`
	ClientID     string   `flag:"client-id" usage:"OAuth2 client ID" required:"true"`
	ClientSecret string   `flag:"client-secret" usage:"OAuth2 client secret" required:"true"`
	Scopes       []string `flag:"scope" usage:"OAuth2 scopes (can be specified multiple times)"`
	ClaimHeader  []string `flag:"claim-header" usage:"Claim to header mapping in format 'claim:header' (e.g., 'email:X-User-Email')"`
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

	// Parse claim mappings
	var claimMappings []ingress_v1alpha.ClaimMappings
	for _, mapping := range opts.ClaimHeader {
		parts := strings.SplitN(mapping, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid claim-header mapping format: %s (expected 'claim:header')", mapping)
		}
		claimMappings = append(claimMappings, ingress_v1alpha.ClaimMappings{
			Claim:  strings.TrimSpace(parts[0]),
			Header: strings.TrimSpace(parts[1]),
		})
	}

	// Build scopes string
	scopes := "openid"
	if len(opts.Scopes) > 0 {
		scopes = strings.Join(opts.Scopes, " ")
		// Ensure openid is included
		if !strings.Contains(scopes, "openid") {
			scopes = "openid " + scopes
		}
	}

	// Configure OIDC
	oidcConfig := ingress_v1alpha.OidcConfig{
		ProviderUrl:   opts.ProviderURL,
		ClientId:      opts.ClientID,
		ClientSecret:  opts.ClientSecret,
		Scopes:        scopes,
		ClaimMappings: claimMappings,
	}

	// Update the route with OIDC config
	_, err = ic.UpdateOIDCConfig(ctx, opts.Host, oidcConfig)
	if err != nil {
		return fmt.Errorf("failed to update route OIDC config: %w", err)
	}

	ctx.Printf("OIDC enabled for route: %s\n", opts.Host)
	ctx.Printf("Provider: %s\n", opts.ProviderURL)
	ctx.Printf("Client ID: %s\n", opts.ClientID)
	if len(claimMappings) > 0 {
		ctx.Printf("Claim mappings:\n")
		for _, mapping := range claimMappings {
			ctx.Printf("  %s → %s\n", mapping.Claim, mapping.Header)
		}
	}
	return nil
}
