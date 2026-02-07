package postgresql

import (
	"context"
	"fmt"

	"miren.dev/runtime/api/addon/addon_v1alpha"
	"miren.dev/runtime/pkg/addon"
	"miren.dev/runtime/pkg/entity/types"
	"miren.dev/runtime/pkg/idgen"
)

const postgresPort = 5432

func (p *Provider) provisionDedicated(ctx context.Context, app addon.App, plan addon.Plan) (*addon.ProvisionResult, error) {
	password := idgen.Gen("pw")
	dbName := sanitizeIdentifier(app.Name)
	userName := sanitizeIdentifier(app.Name)
	serviceName := fmt.Sprintf("%s-postgresql", app.Name)

	p.log.Info("provisioning dedicated PostgreSQL",
		"app", app.Name,
		"plan", plan.Name)

	// Create PostgresServer entity
	server := &addon_v1alpha.PostgresServer{
		AddonName:         AddonName,
		Plan:              plan.Name,
		Status:            "provisioning",
		AssociationCount:  1,
		SuperuserPassword: password,
	}

	serverName := fmt.Sprintf("pg-%s-%s", app.Name, idgen.Gen("s"))
	serverID, err := p.fw.EC.Create(ctx, serverName, server)
	if err != nil {
		return nil, fmt.Errorf("creating postgres server entity: %w", err)
	}

	// Create sandbox pool for the PostgreSQL container
	labels := types.LabelSet(
		"addon", AddonName,
		"app", app.Name,
		"server", serverName,
	)

	env := []string{
		"POSTGRES_DB=" + dbName,
		"POSTGRES_USER=" + userName,
		"POSTGRES_PASSWORD=" + password,
	}

	poolID, err := p.fw.CreateSandboxPool(ctx, addon.CreateSandboxPoolSpec{
		Name:             serverName,
		Image:            DefaultImage,
		Env:              env,
		DesiredInstances: 1,
		Labels:           labels,
		SandboxPrefix:    fmt.Sprintf("%s-pg", app.Name),
	})
	if err != nil {
		// Cleanup: delete server entity
		_ = p.fw.EC.Delete(ctx, serverID)
		return nil, fmt.Errorf("creating sandbox pool: %w", err)
	}

	// Wait for pool to have a running instance
	if err := p.fw.WaitForPool(ctx, poolID, poolReadyTimeout); err != nil {
		_ = p.fw.DeleteSandboxPool(ctx, poolID)
		_ = p.fw.EC.Delete(ctx, serverID)
		return nil, fmt.Errorf("waiting for postgres pool: %w", err)
	}

	// Create service for network access
	svcID, err := p.fw.CreateService(ctx, serviceName, labels, postgresPort)
	if err != nil {
		_ = p.fw.DeleteSandboxPool(ctx, poolID)
		_ = p.fw.EC.Delete(ctx, serverID)
		return nil, fmt.Errorf("creating service: %w", err)
	}

	// Update server with pool and service refs
	server.SandboxPool = poolID
	server.Service = svcID
	server.Status = "active"
	server.ID = serverID

	if err := p.fw.EC.Update(ctx, server); err != nil {
		_ = p.fw.DeleteService(ctx, svcID)
		_ = p.fw.DeleteSandboxPool(ctx, poolID)
		_ = p.fw.EC.Delete(ctx, serverID)
		return nil, fmt.Errorf("updating postgres server: %w", err)
	}

	p.log.Info("dedicated PostgreSQL provisioned",
		"server", serverName,
		"pool", poolID,
		"service", svcID)

	host := serviceName + ".addon.app.miren"
	envVars := buildEnvVars(host, postgresPort, userName, password, dbName)

	// Store the dedicated data as attrs on the association
	dedicatedData := &addon_v1alpha.PostgresqlDedicatedData{
		PostgresServer: serverID,
	}

	return &addon.ProvisionResult{
		EnvVars: envVars,
		Attrs:   dedicatedData.Encode(),
	}, nil
}

func (p *Provider) deprovisionDedicated(ctx context.Context, assoc addon.AddonAssociation) error {
	// Decode the dedicated data from the association
	var dedicatedData addon_v1alpha.PostgresqlDedicatedData
	if assoc.Entity != nil {
		dedicatedData.Decode(assoc.Entity)
	}

	if dedicatedData.PostgresServer == "" {
		p.log.Warn("no postgres server ref found, skipping deprovision")
		return nil
	}

	// Fetch the server entity
	var server addon_v1alpha.PostgresServer
	if err := p.fw.EC.GetById(ctx, dedicatedData.PostgresServer, &server); err != nil {
		p.log.Warn("failed to get postgres server for deprovision", "error", err)
		// Server may already be deleted, continue with best-effort cleanup
		return nil
	}

	// Delete service
	if server.Service != "" {
		if err := p.fw.DeleteService(ctx, server.Service); err != nil {
			p.log.Warn("failed to delete service", "error", err)
		}
	}

	// Delete sandbox pool
	if server.SandboxPool != "" {
		if err := p.fw.DeleteSandboxPool(ctx, server.SandboxPool); err != nil {
			p.log.Warn("failed to delete sandbox pool", "error", err)
		}
	}

	// Delete server entity
	if err := p.fw.EC.Delete(ctx, server.ID); err != nil {
		p.log.Warn("failed to delete postgres server entity", "error", err)
	}

	p.log.Info("dedicated PostgreSQL deprovisioned", "server", server.ID)
	return nil
}

// sanitizeIdentifier ensures a name is safe for use as a PostgreSQL identifier.
func sanitizeIdentifier(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			result = append(result, c)
		} else if c >= 'A' && c <= 'Z' {
			result = append(result, c+32) // lowercase
		} else if c == '-' {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "app"
	}
	// Ensure it starts with a letter
	if result[0] >= '0' && result[0] <= '9' {
		result = append([]byte{'a'}, result...)
	}
	return string(result)
}
