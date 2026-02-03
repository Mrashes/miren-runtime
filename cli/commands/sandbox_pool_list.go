package commands

import (
	"fmt"
	"time"

	"miren.dev/runtime/api/compute/compute_v1alpha"
	"miren.dev/runtime/api/core/core_v1alpha"
	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/pkg/ui"
)

func SandboxPoolList(ctx *Context, opts struct {
	FormatOptions
	ConfigCentric
}) error {
	client, err := ctx.RPCClient("entities")
	if err != nil {
		return err
	}

	eac := entityserver_v1alpha.NewEntityAccessClient(client)

	kindRes, err := eac.LookupKind(ctx, "sandbox_pool")
	if err != nil {
		return err
	}

	res, err := eac.List(ctx, kindRes.Attr())
	if err != nil {
		return err
	}

	// Fetch app versions to determine concurrency mode
	versionKindRes, err := eac.LookupKind(ctx, "app_version")
	if err != nil {
		return err
	}

	versionsRes, err := eac.List(ctx, versionKindRes.Attr())
	if err != nil {
		return err
	}

	// Build version map for lookup
	versionMap := make(map[string]*core_v1alpha.AppVersion)
	for _, e := range versionsRes.Values() {
		v := new(core_v1alpha.AppVersion)
		v.Decode(e.Entity())
		versionMap[v.ID.String()] = v
	}

	if opts.IsJSON() {
		var pools []compute_v1alpha.SandboxPool

		for _, e := range res.Values() {
			var pool compute_v1alpha.SandboxPool
			pool.Decode(e.Entity())
			pools = append(pools, pool)
		}

		return PrintJSON(pools)
	}

	var rows []ui.Row
	headers := []string{"ID", "VERSION", "SERVICE", "MODE", "DESIRED", "CURRENT", "READY", "CREATED", "UPDATED"}

	for _, e := range res.Values() {
		var pool compute_v1alpha.SandboxPool
		pool.Decode(e.Entity())

		// Determine scaling mode from version config
		mode := "auto"
		if version, ok := versionMap[pool.SandboxSpec.Version.String()]; ok {
			for _, svc := range version.Config.Services {
				if svc.Name == pool.Service && svc.ServiceConcurrency.Mode == "fixed" {
					mode = "fixed"
					break
				}
			}
		}

		rows = append(rows, ui.Row{
			ui.CleanEntityID(pool.ID.String()),
			ui.DisplayAppVersion(pool.SandboxSpec.Version.String()),
			pool.Service,
			mode,
			fmt.Sprintf("%d", pool.DesiredInstances),
			fmt.Sprintf("%d", pool.CurrentInstances),
			fmt.Sprintf("%d", pool.ReadyInstances),
			humanFriendlyTimestamp(time.UnixMilli(e.CreatedAt())),
			humanFriendlyTimestamp(time.UnixMilli(e.UpdatedAt())),
		})
	}

	if len(rows) == 0 {
		ctx.Printf("No sandbox pools found\n")
		return nil
	}

	columns := ui.AutoSizeColumns(headers, rows, ui.Columns().NoTruncate(0))
	table := ui.NewTable(
		ui.WithColumns(columns),
		ui.WithRows(rows),
	)

	ctx.Printf("%s\n", table.Render())
	return nil
}
