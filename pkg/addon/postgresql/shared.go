package postgresql

import (
	"context"
	"fmt"
	"time"

	"miren.dev/runtime/api/addon/addon_v1alpha"
	"miren.dev/runtime/pkg/addon"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/entity/types"
	"miren.dev/runtime/pkg/idgen"
)

const (
	sharedServerName = "pg-shared"
	poolReadyTimeout = 5 * time.Minute
)

func (p *Provider) provisionShared(ctx context.Context, app addon.App, plan addon.Plan) (*addon.ProvisionResult, error) {
	password := idgen.Gen("pw")
	dbName := sanitizeIdentifier(app.Name)
	userName := sanitizeIdentifier(app.Name)

	p.log.Info("provisioning shared PostgreSQL",
		"app", app.Name,
		"plan", plan.Name)

	// Find or create the shared server
	serverID, server, err := p.findOrCreateSharedServer(ctx)
	if err != nil {
		return nil, fmt.Errorf("ensuring shared server: %w", err)
	}

	// Increment association count
	server.AssociationCount++
	server.ID = serverID
	if err := p.fw.EC.Update(ctx, server); err != nil {
		return nil, fmt.Errorf("updating shared server association count: %w", err)
	}

	// TODO: Connect to the shared PostgreSQL server and CREATE USER/DATABASE.
	// For now, we set up the env vars pointing to the shared server,
	// and the superuser credentials are used by the CREATE commands.
	// The actual SQL DDL (CREATE USER, CREATE DATABASE) will be added
	// when we have network connectivity to the postgres sandbox.

	p.log.Info("shared PostgreSQL provisioned",
		"app", app.Name,
		"server", serverID,
		"database", dbName)

	serviceName := sharedServerName + "-postgresql"
	host := serviceName + ".addon.app.miren"
	envVars := buildEnvVars(host, postgresPort, userName, password, dbName)

	sharedData := &addon_v1alpha.PostgresqlSharedData{
		PostgresServer: serverID,
		DatabaseName:   dbName,
		Username:       userName,
	}

	return &addon.ProvisionResult{
		EnvVars: envVars,
		Attrs:   sharedData.Encode(),
	}, nil
}

func (p *Provider) findOrCreateSharedServer(ctx context.Context) (entity.Id, *addon_v1alpha.PostgresServer, error) {
	// Try to find an existing shared server
	var server addon_v1alpha.PostgresServer
	err := p.fw.EC.Get(ctx, sharedServerName, &server)
	if err == nil && server.Status == "active" {
		return server.ID, &server, nil
	}

	// Create a new shared server
	p.log.Info("creating new shared PostgreSQL server")

	superuserPassword := idgen.Gen("su")

	newServer := &addon_v1alpha.PostgresServer{
		AddonName:         AddonName,
		Plan:              "shared",
		Status:            "provisioning",
		AssociationCount:  0,
		SuperuserPassword: superuserPassword,
	}

	serverID, err := p.fw.EC.Create(ctx, sharedServerName, newServer)
	if err != nil {
		return "", nil, fmt.Errorf("creating shared server entity: %w", err)
	}

	// Create sandbox pool
	labels := types.LabelSet(
		"addon", AddonName,
		"server", sharedServerName,
		"shared", "true",
	)

	env := []string{
		"POSTGRES_PASSWORD=" + superuserPassword,
	}

	poolID, err := p.fw.CreateSandboxPool(ctx, addon.CreateSandboxPoolSpec{
		Name:             sharedServerName,
		Image:            DefaultImage,
		Env:              env,
		DesiredInstances: 1,
		Labels:           labels,
		SandboxPrefix:    "pg-shared",
	})
	if err != nil {
		_ = p.fw.EC.Delete(ctx, serverID)
		return "", nil, fmt.Errorf("creating shared pool: %w", err)
	}

	// Wait for pool
	if err := p.fw.WaitForPool(ctx, poolID, poolReadyTimeout); err != nil {
		_ = p.fw.DeleteSandboxPool(ctx, poolID)
		_ = p.fw.EC.Delete(ctx, serverID)
		return "", nil, fmt.Errorf("waiting for shared pool: %w", err)
	}

	// Create service
	serviceName := sharedServerName + "-postgresql"
	svcID, err := p.fw.CreateService(ctx, serviceName, labels, postgresPort)
	if err != nil {
		_ = p.fw.DeleteSandboxPool(ctx, poolID)
		_ = p.fw.EC.Delete(ctx, serverID)
		return "", nil, fmt.Errorf("creating shared service: %w", err)
	}

	// Update server
	newServer.SandboxPool = poolID
	newServer.Service = svcID
	newServer.Status = "active"
	newServer.ID = serverID

	if err := p.fw.EC.Update(ctx, newServer); err != nil {
		_ = p.fw.DeleteService(ctx, svcID)
		_ = p.fw.DeleteSandboxPool(ctx, poolID)
		_ = p.fw.EC.Delete(ctx, serverID)
		return "", nil, fmt.Errorf("updating shared server: %w", err)
	}

	p.log.Info("shared PostgreSQL server created",
		"server", serverID,
		"pool", poolID,
		"service", svcID)

	return serverID, newServer, nil
}

func (p *Provider) deprovisionShared(ctx context.Context, assoc addon.AddonAssociation) error {
	var sharedData addon_v1alpha.PostgresqlSharedData
	if assoc.Entity != nil {
		sharedData.Decode(assoc.Entity)
	}

	if sharedData.PostgresServer == "" {
		p.log.Warn("no postgres server ref found, skipping deprovision")
		return nil
	}

	// Fetch the server
	var server addon_v1alpha.PostgresServer
	if err := p.fw.EC.GetById(ctx, sharedData.PostgresServer, &server); err != nil {
		p.log.Warn("failed to get shared server for deprovision", "error", err)
		return nil
	}

	// TODO: Connect to the shared server and DROP USER/DATABASE
	// Similar to provisioning, the actual SQL DDL will be added later.

	// Decrement association count
	server.AssociationCount--
	if err := p.fw.EC.Update(ctx, &server); err != nil {
		p.log.Warn("failed to update association count", "error", err)
	}

	// If no more associations, tear down the shared server
	if server.AssociationCount <= 0 {
		p.log.Info("last association removed, tearing down shared server", "server", server.ID)

		if server.Service != "" {
			if err := p.fw.DeleteService(ctx, server.Service); err != nil {
				p.log.Warn("failed to delete shared service", "error", err)
			}
		}

		if server.SandboxPool != "" {
			if err := p.fw.DeleteSandboxPool(ctx, server.SandboxPool); err != nil {
				p.log.Warn("failed to delete shared pool", "error", err)
			}
		}

		if err := p.fw.EC.Delete(ctx, server.ID); err != nil {
			p.log.Warn("failed to delete shared server entity", "error", err)
		}
	}

	p.log.Info("shared PostgreSQL deprovisioned",
		"app_database", sharedData.DatabaseName,
		"remaining_associations", server.AssociationCount)

	return nil
}
