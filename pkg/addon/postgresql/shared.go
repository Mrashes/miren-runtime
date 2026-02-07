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
	"miren.dev/runtime/pkg/saga"
)

const (
	sharedServerName = "pg-shared"
	poolReadyTimeout = 5 * time.Minute
)

// --- Shared Provisioning Saga Actions ---

// Step 1: Find or create the shared PostgresServer.
// If no active shared server exists, this creates the server entity,
// sandbox pool, service, and activates it (inline EnsureSharedServerSaga).

type FindOrCreateSharedServerIn struct {
	AppName string
}

type FindOrCreateSharedServerOut struct {
	ServerID          entity.Id
	SuperuserPassword string
}

func FindOrCreateSharedServer(ctx context.Context, in FindOrCreateSharedServerIn) (FindOrCreateSharedServerOut, error) {
	fw := saga.Get[*addon.ProviderFramework](ctx)

	// Try to find an existing active shared server
	var server addon_v1alpha.PostgresServer
	err := fw.EC.Get(ctx, sharedServerName, &server)
	if err == nil && server.Status == "active" {
		return FindOrCreateSharedServerOut{
			ServerID:          server.ID,
			SuperuserPassword: server.SuperuserPassword,
		}, nil
	}

	// No active shared server exists — run EnsureSharedServerSaga inline.
	superuserPassword := idgen.Gen("su")

	newServer := &addon_v1alpha.PostgresServer{
		AddonName:         AddonName,
		Plan:              "shared",
		Status:            "provisioning",
		AssociationCount:  0,
		SuperuserPassword: superuserPassword,
	}

	serverID, err := fw.EC.Create(ctx, sharedServerName, newServer)
	if err != nil {
		return FindOrCreateSharedServerOut{}, fmt.Errorf("creating shared server entity: %w", err)
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

	poolID, err := fw.CreateSandboxPool(ctx, addon.CreateSandboxPoolSpec{
		Name:             sharedServerName,
		Image:            DefaultImage,
		Env:              env,
		DesiredInstances: 1,
		Labels:           labels,
		SandboxPrefix:    "pg-shared",
	})
	if err != nil {
		_ = fw.EC.Delete(ctx, serverID)
		return FindOrCreateSharedServerOut{}, fmt.Errorf("creating shared pool: %w", err)
	}

	// Wait for pool to be ready
	if err := fw.WaitForPool(ctx, poolID, poolReadyTimeout); err != nil {
		_ = fw.DeleteSandboxPool(ctx, poolID)
		_ = fw.EC.Delete(ctx, serverID)
		return FindOrCreateSharedServerOut{}, fmt.Errorf("waiting for shared pool: %w", err)
	}

	// Create service
	serviceName := sharedServerName + "-postgresql"
	svcID, err := fw.CreateService(ctx, serviceName, labels, postgresPort)
	if err != nil {
		_ = fw.DeleteSandboxPool(ctx, poolID)
		_ = fw.EC.Delete(ctx, serverID)
		return FindOrCreateSharedServerOut{}, fmt.Errorf("creating shared service: %w", err)
	}

	// Activate the server
	newServer.SandboxPool = poolID
	newServer.Service = svcID
	newServer.Status = "active"
	newServer.ID = serverID

	if err := fw.EC.Update(ctx, newServer); err != nil {
		_ = fw.DeleteService(ctx, svcID)
		_ = fw.DeleteSandboxPool(ctx, poolID)
		_ = fw.EC.Delete(ctx, serverID)
		return FindOrCreateSharedServerOut{}, fmt.Errorf("activating shared server: %w", err)
	}

	return FindOrCreateSharedServerOut{
		ServerID:          serverID,
		SuperuserPassword: superuserPassword,
	}, nil
}

func UndoFindOrCreateSharedServer(ctx context.Context, in FindOrCreateSharedServerIn, out FindOrCreateSharedServerOut) error {
	// The shared server is intentionally not torn down if a later provisioning
	// step fails — it may be serving other applications. The EnsureSharedServerSaga
	// handles its own compensations inline above.
	return nil
}

// Step 2: Generate credentials for the app's database.

type GenerateSharedCredentialsIn struct {
	AppName string
}

type GenerateSharedCredentialsOut struct {
	SharedPassword     string
	SharedDatabaseName string
	SharedUsername     string
}

func GenerateSharedCredentials(ctx context.Context, in GenerateSharedCredentialsIn) (GenerateSharedCredentialsOut, error) {
	return GenerateSharedCredentialsOut{
		SharedPassword:     idgen.Gen("pw"),
		SharedDatabaseName: sanitizeIdentifier(in.AppName),
		SharedUsername:     sanitizeIdentifier(in.AppName),
	}, nil
}

func UndoGenerateSharedCredentials(ctx context.Context, in GenerateSharedCredentialsIn, out GenerateSharedCredentialsOut) error {
	return nil
}

// Step 3: Connect to the shared server and CREATE USER.

type CreateSharedUserIn struct {
	ServerID          entity.Id
	SuperuserPassword string
	SharedUsername    string
	SharedPassword    string
}

type CreateSharedUserOut struct {
	UserCreated bool
}

func CreateSharedUser(ctx context.Context, in CreateSharedUserIn) (CreateSharedUserOut, error) {
	// TODO: Connect to the shared PostgreSQL server via the Service endpoint
	// and execute: CREATE USER {username} WITH PASSWORD '{password}'
	//
	// This requires network connectivity to the postgres sandbox, which will
	// be available once the Service entity is routing traffic.
	// For now, the credentials are set up but the SQL DDL is deferred.
	return CreateSharedUserOut{UserCreated: true}, nil
}

func UndoCreateSharedUser(ctx context.Context, in CreateSharedUserIn, out CreateSharedUserOut) error {
	if !out.UserCreated {
		return nil
	}
	// TODO: Connect and execute DROP USER {username}
	return nil
}

// Step 4: Connect to the shared server and CREATE DATABASE.

type CreateSharedDatabaseIn struct {
	ServerID          entity.Id
	SuperuserPassword string
	SharedDatabaseName string
	SharedUsername     string
}

type CreateSharedDatabaseOut struct {
	DatabaseCreated bool
}

func CreateSharedDatabase(ctx context.Context, in CreateSharedDatabaseIn) (CreateSharedDatabaseOut, error) {
	// TODO: Connect to the shared PostgreSQL server via the Service endpoint
	// and execute: CREATE DATABASE {dbname} OWNER {username}
	//
	// This requires network connectivity to the postgres sandbox.
	// For now, the database name is recorded but the SQL DDL is deferred.
	return CreateSharedDatabaseOut{DatabaseCreated: true}, nil
}

func UndoCreateSharedDatabase(ctx context.Context, in CreateSharedDatabaseIn, out CreateSharedDatabaseOut) error {
	if !out.DatabaseCreated {
		return nil
	}
	// TODO: Connect and execute DROP DATABASE {dbname}
	return nil
}

// Step 5: Increment association_count on the shared PostgresServer.

type IncrementAssociationCountIn struct {
	ServerID entity.Id
}

type IncrementAssociationCountOut struct {
	Incremented bool
}

func IncrementAssociationCount(ctx context.Context, in IncrementAssociationCountIn) (IncrementAssociationCountOut, error) {
	fw := saga.Get[*addon.ProviderFramework](ctx)

	var server addon_v1alpha.PostgresServer
	if err := fw.EC.GetById(ctx, in.ServerID, &server); err != nil {
		return IncrementAssociationCountOut{}, fmt.Errorf("getting server for count increment: %w", err)
	}

	server.AssociationCount++
	if err := fw.EC.Update(ctx, &server); err != nil {
		return IncrementAssociationCountOut{}, fmt.Errorf("updating association count: %w", err)
	}

	return IncrementAssociationCountOut{Incremented: true}, nil
}

func UndoIncrementAssociationCount(ctx context.Context, in IncrementAssociationCountIn, out IncrementAssociationCountOut) error {
	if !out.Incremented {
		return nil
	}

	fw := saga.Get[*addon.ProviderFramework](ctx)

	var server addon_v1alpha.PostgresServer
	if err := fw.EC.GetById(ctx, in.ServerID, &server); err != nil {
		return err
	}

	server.AssociationCount--
	return fw.EC.Update(ctx, &server)
}

// Step 6: Build the ProvisionResult.

type BuildSharedResultIn struct {
	ServerID           entity.Id
	SharedDatabaseName string
	SharedUsername     string
	SharedPassword     string
}

type BuildSharedResultOut struct {
	Done bool
}

func BuildSharedResult(ctx context.Context, in BuildSharedResultIn) (BuildSharedResultOut, error) {
	rc := saga.Get[*resultCapture](ctx)

	serviceName := sharedServerName + "-postgresql"
	host := serviceName + ".addon.app.miren"
	envVars := buildEnvVars(host, postgresPort, in.SharedUsername, in.SharedPassword, in.SharedDatabaseName)

	sharedData := &addon_v1alpha.PostgresqlSharedData{
		PostgresServer: in.ServerID,
		DatabaseName:   in.SharedDatabaseName,
		Username:       in.SharedUsername,
	}

	rc.Result = &addon.ProvisionResult{
		EnvVars: envVars,
		Attrs:   sharedData.Encode(),
	}

	return BuildSharedResultOut{Done: true}, nil
}

func UndoBuildSharedResult(ctx context.Context, in BuildSharedResultIn, out BuildSharedResultOut) error {
	return nil
}

// RegisterSharedSaga registers the shared PostgreSQL provisioning saga.
func RegisterSharedSaga(registry *saga.Registry, fw *addon.ProviderFramework, rc *resultCapture) error {
	return saga.Define("provision-shared-postgresql").
		Using(fw).
		Using(rc).
		Action(FindOrCreateSharedServer).Undo(UndoFindOrCreateSharedServer).
		Action(GenerateSharedCredentials).Undo(UndoGenerateSharedCredentials).
		Action(CreateSharedUser).Undo(UndoCreateSharedUser).
		Action(CreateSharedDatabase).Undo(UndoCreateSharedDatabase).
		Action(IncrementAssociationCount).Undo(UndoIncrementAssociationCount).
		Action(BuildSharedResult).Undo(UndoBuildSharedResult).
		RegisterTo(registry)
}

// --- Shared Deprovisioning Saga Actions ---

type DecodeSharedAttrsIn struct {
	AssocEntity *entity.Entity `saga:"assocentity"`
}

type DecodeSharedAttrsOut struct {
	SharedServerRef  entity.Id
	SharedDbName     string
	SharedUserName   string
}

func DecodeSharedAttrs(ctx context.Context, in DecodeSharedAttrsIn) (DecodeSharedAttrsOut, error) {
	var data addon_v1alpha.PostgresqlSharedData
	if in.AssocEntity != nil {
		data.Decode(in.AssocEntity)
	}

	if data.PostgresServer == "" {
		return DecodeSharedAttrsOut{}, fmt.Errorf("no postgres server ref found")
	}

	return DecodeSharedAttrsOut{
		SharedServerRef:  data.PostgresServer,
		SharedDbName:     data.DatabaseName,
		SharedUserName:   data.Username,
	}, nil
}

func UndoDecodeSharedAttrs(ctx context.Context, in DecodeSharedAttrsIn, out DecodeSharedAttrsOut) error {
	return nil
}

type LookupSharedServerIn struct {
	SharedServerRef entity.Id
}

type LookupSharedServerOut struct {
	SharedSuperuserPassword string
	SharedServiceRef        entity.Id
	SharedPoolRef           entity.Id
	SharedAssocCount        int64
}

func LookupSharedServer(ctx context.Context, in LookupSharedServerIn) (LookupSharedServerOut, error) {
	fw := saga.Get[*addon.ProviderFramework](ctx)

	var server addon_v1alpha.PostgresServer
	if err := fw.EC.GetById(ctx, in.SharedServerRef, &server); err != nil {
		return LookupSharedServerOut{}, fmt.Errorf("looking up shared server: %w", err)
	}

	return LookupSharedServerOut{
		SharedSuperuserPassword: server.SuperuserPassword,
		SharedServiceRef:        server.Service,
		SharedPoolRef:           server.SandboxPool,
		SharedAssocCount:        server.AssociationCount,
	}, nil
}

func UndoLookupSharedServer(ctx context.Context, in LookupSharedServerIn, out LookupSharedServerOut) error {
	return nil
}

type TerminateConnectionsIn struct {
	SharedServerRef         entity.Id
	SharedSuperuserPassword string
	SharedDbName            string
}

type TerminateConnectionsOut struct {
	Done bool
}

func TerminateConnections(ctx context.Context, in TerminateConnectionsIn) (TerminateConnectionsOut, error) {
	// TODO: Connect to the shared server and execute:
	// SELECT pg_terminate_backend(pid)
	// FROM pg_stat_activity
	// WHERE datname = '{dbname}' AND pid <> pg_backend_pid()
	return TerminateConnectionsOut{Done: true}, nil
}

func UndoTerminateConnections(ctx context.Context, in TerminateConnectionsIn, out TerminateConnectionsOut) error {
	return nil
}

type DropSharedDatabaseIn struct {
	SharedServerRef         entity.Id
	SharedSuperuserPassword string
	SharedDbName            string
}

type DropSharedDatabaseOut struct {
	DatabaseDropped bool
}

func DropSharedDatabase(ctx context.Context, in DropSharedDatabaseIn) (DropSharedDatabaseOut, error) {
	// TODO: Connect to the shared server and execute:
	// DROP DATABASE {dbname}
	return DropSharedDatabaseOut{DatabaseDropped: true}, nil
}

func UndoDropSharedDatabase(ctx context.Context, in DropSharedDatabaseIn, out DropSharedDatabaseOut) error {
	return nil
}

type DropSharedUserIn struct {
	SharedServerRef         entity.Id
	SharedSuperuserPassword string
	SharedUserName          string
}

type DropSharedUserOut struct {
	UserDropped bool
}

func DropSharedUser(ctx context.Context, in DropSharedUserIn) (DropSharedUserOut, error) {
	// TODO: Connect to the shared server and execute:
	// DROP USER {username}
	return DropSharedUserOut{UserDropped: true}, nil
}

func UndoDropSharedUser(ctx context.Context, in DropSharedUserIn, out DropSharedUserOut) error {
	return nil
}

type DecrementAssociationCountIn struct {
	SharedServerRef entity.Id
}

type DecrementAssociationCountOut struct {
	RemainingCount int64
}

func DecrementAssociationCount(ctx context.Context, in DecrementAssociationCountIn) (DecrementAssociationCountOut, error) {
	fw := saga.Get[*addon.ProviderFramework](ctx)

	var server addon_v1alpha.PostgresServer
	if err := fw.EC.GetById(ctx, in.SharedServerRef, &server); err != nil {
		return DecrementAssociationCountOut{}, fmt.Errorf("getting server: %w", err)
	}

	server.AssociationCount--
	if err := fw.EC.Update(ctx, &server); err != nil {
		return DecrementAssociationCountOut{}, fmt.Errorf("updating association count: %w", err)
	}

	return DecrementAssociationCountOut{RemainingCount: server.AssociationCount}, nil
}

func UndoDecrementAssociationCount(ctx context.Context, in DecrementAssociationCountIn, out DecrementAssociationCountOut) error {
	return nil
}

type CleanupSharedServerIn struct {
	SharedServerRef  entity.Id
	SharedServiceRef entity.Id
	SharedPoolRef    entity.Id
	RemainingCount   int64
}

type CleanupSharedServerOut struct {
	CleanedUp bool
}

func CleanupSharedServer(ctx context.Context, in CleanupSharedServerIn) (CleanupSharedServerOut, error) {
	if in.RemainingCount > 0 {
		return CleanupSharedServerOut{CleanedUp: false}, nil
	}

	fw := saga.Get[*addon.ProviderFramework](ctx)

	if in.SharedServiceRef != "" {
		if err := fw.DeleteService(ctx, in.SharedServiceRef); err != nil {
			return CleanupSharedServerOut{}, fmt.Errorf("deleting shared service: %w", err)
		}
	}

	if in.SharedPoolRef != "" {
		if err := fw.DeleteSandboxPool(ctx, in.SharedPoolRef); err != nil {
			return CleanupSharedServerOut{}, fmt.Errorf("deleting shared pool: %w", err)
		}
	}

	if err := fw.EC.Delete(ctx, in.SharedServerRef); err != nil {
		return CleanupSharedServerOut{}, fmt.Errorf("deleting shared server: %w", err)
	}

	return CleanupSharedServerOut{CleanedUp: true}, nil
}

func UndoCleanupSharedServer(ctx context.Context, in CleanupSharedServerIn, out CleanupSharedServerOut) error {
	return nil
}

// RegisterDeprovisionSharedSaga registers the shared deprovisioning saga.
func RegisterDeprovisionSharedSaga(registry *saga.Registry, fw *addon.ProviderFramework) error {
	return saga.Define("deprovision-shared-postgresql").
		Using(fw).
		Action(DecodeSharedAttrs).Undo(UndoDecodeSharedAttrs).
		Action(LookupSharedServer).Undo(UndoLookupSharedServer).
		Action(TerminateConnections).Undo(UndoTerminateConnections).
		Action(DropSharedDatabase).Undo(UndoDropSharedDatabase).
		Action(DropSharedUser).Undo(UndoDropSharedUser).
		Action(DecrementAssociationCount).Undo(UndoDecrementAssociationCount).
		Action(CleanupSharedServer).Undo(UndoCleanupSharedServer).
		RegisterTo(registry)
}

func (p *Provider) provisionShared(ctx context.Context, app addon.App, plan addon.Plan) (*addon.ProvisionResult, error) {
	p.log.Info("provisioning shared PostgreSQL",
		"app", app.Name,
		"plan", plan.Name)

	rc := &resultCapture{}
	registry := saga.NewRegistry()

	if err := RegisterSharedSaga(registry, p.fw, rc); err != nil {
		return nil, fmt.Errorf("registering shared saga: %w", err)
	}

	storage := saga.NewMemoryStorage()
	executor := saga.NewExecutor(storage, saga.WithRegistry(registry))

	err := executor.Start("provision-shared-postgresql").
		Input("appname", app.Name).
		Execute(ctx)
	if err != nil {
		return nil, err
	}

	if rc.Result == nil {
		return nil, fmt.Errorf("saga completed but no result was captured")
	}

	p.log.Info("shared PostgreSQL provisioned", "app", app.Name)
	return rc.Result, nil
}

func (p *Provider) deprovisionShared(ctx context.Context, assoc addon.AddonAssociation) error {
	p.log.Info("deprovisioning shared PostgreSQL", "assoc", assoc.ID)

	registry := saga.NewRegistry()

	if err := RegisterDeprovisionSharedSaga(registry, p.fw); err != nil {
		return fmt.Errorf("registering deprovision saga: %w", err)
	}

	storage := saga.NewMemoryStorage()
	executor := saga.NewExecutor(storage, saga.WithRegistry(registry))

	err := executor.Start("deprovision-shared-postgresql").
		Input("assocentity", assoc.Entity).
		Execute(ctx)
	if err != nil {
		return err
	}

	p.log.Info("shared PostgreSQL deprovisioned", "assoc", assoc.ID)
	return nil
}
