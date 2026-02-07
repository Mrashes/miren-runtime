package postgresql

import (
	"context"
	"fmt"
	"log/slog"

	"miren.dev/runtime/pkg/addon"
)

// Provider implements the AddonProvider interface for PostgreSQL.
type Provider struct {
	fw  *addon.ProviderFramework
	log *slog.Logger
}

// NewProvider creates a new PostgreSQL addon provider.
func NewProvider(fw *addon.ProviderFramework) *Provider {
	return &Provider{
		fw:  fw,
		log: fw.Log.With("addon", AddonName),
	}
}

func (p *Provider) Provision(ctx context.Context, app addon.App, plan addon.Plan) (*addon.ProvisionResult, error) {
	if IsSharedPlan(plan.Name) {
		return p.provisionShared(ctx, app, plan)
	}
	return p.provisionDedicated(ctx, app, plan)
}

func (p *Provider) AdjustEnvVars(ctx context.Context, result *addon.ProvisionResult, assoc addon.AddonAssociation, collisions []string) ([]addon.Variable, error) {
	// For PostgreSQL, we don't adjust variable names on collision.
	// The addon's vars take priority and the user should rename their
	// conflicting manual vars instead.
	return result.EnvVars, nil
}

func (p *Provider) Deprovision(ctx context.Context, assoc addon.AddonAssociation) error {
	plan := assoc.Plan
	if IsSharedPlan(plan) {
		return p.deprovisionShared(ctx, assoc)
	}
	return p.deprovisionDedicated(ctx, assoc)
}

// buildDatabaseURL constructs a postgres:// connection URL.
func buildDatabaseURL(host string, port int, user, password, database string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", user, password, host, port, database)
}

// buildEnvVars creates the standard set of PostgreSQL environment variables.
func buildEnvVars(host string, port int, user, password, database string) []addon.Variable {
	return []addon.Variable{
		{Key: "DATABASE_URL", Value: buildDatabaseURL(host, port, user, password, database), Sensitive: true},
		{Key: "PGHOST", Value: host},
		{Key: "PGPORT", Value: fmt.Sprintf("%d", port)},
		{Key: "PGUSER", Value: user},
		{Key: "PGPASSWORD", Value: password, Sensitive: true},
		{Key: "PGDATABASE", Value: database},
	}
}
