package entity

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// MigrateShortIdOptions configures the short-id migration behavior.
type MigrateShortIdOptions struct {
	DryRun bool
	Prefix string
}

// MigrateShortIds backfills db/short-id for all entities that don't have one.
// This is idempotent — entities that already have a short-id are skipped.
func MigrateShortIds(ctx context.Context, log *slog.Logger, client *clientv3.Client, opts MigrateShortIdOptions) (migrated int, skipped int, err error) {
	prefix := path.Join(opts.Prefix, "entity")
	refsPrefix := path.Join(opts.Prefix, "refs") + "/"

	log.Info("starting short-id migration", "prefix", prefix, "dry_run", opts.DryRun)

	resp, err := client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list entities: %w", err)
	}

	log.Info("found entities to scan for short-id migration", "count", len(resp.Kvs))

	// Build a set of existing refs for collision checking
	refsResp, err := client.Get(ctx, refsPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list existing refs: %w", err)
	}

	existingRefs := make(map[string]struct{})
	for _, kv := range refsResp.Kvs {
		ref := strings.TrimPrefix(string(kv.Key), refsPrefix)
		existingRefs[ref] = struct{}{}
	}

	var errorCount int

	for _, kv := range resp.Kvs {
		key := string(kv.Key)

		// Skip session keys
		if strings.Contains(key, "/session/") {
			continue
		}

		var ent Entity
		if err := Decode(kv.Value, &ent); err != nil {
			log.Warn("failed to decode entity during short-id migration", "key", key, "error", err)
			errorCount++
			continue
		}

		// Skip entities that already have a short-id
		if ent.ShortId() != "" {
			skipped++
			continue
		}

		// Skip system/schema entities (no entity/kind attribute)
		if _, hasKind := ent.Get(EntityKind); !hasKind {
			skipped++
			continue
		}

		entityId := string(ent.Id())
		if entityId == "" {
			skipped++
			continue
		}

		// Allocate short-id using in-memory set for collision checking
		shortId, allocErr := AllocateShortId(entityId, func(candidate string) (bool, error) {
			_, exists := existingRefs[candidate]
			return exists, nil
		})
		if allocErr != nil {
			log.Warn("failed to allocate short-id", "entity", entityId, "error", allocErr)
			errorCount++
			continue
		}

		if opts.DryRun {
			log.Info("dry-run: would assign short-id", "entity", entityId, "short_id", shortId)
			migrated++
			existingRefs[shortId] = struct{}{}
			continue
		}

		// Set the short-id on the entity
		ent.Set(String(DBShortId, shortId))

		newData, encErr := Encode(&ent)
		if encErr != nil {
			log.Warn("failed to encode entity with short-id", "entity", entityId, "error", encErr)
			errorCount++
			continue
		}

		// Write the updated entity and ref index entry atomically
		refKey := refsPrefix + shortId
		txnResp, txnErr := client.Txn(ctx).
			If(clientv3.Compare(clientv3.CreateRevision(refKey), "=", 0)).
			Then(
				clientv3.OpPut(key, string(newData)),
				clientv3.OpPut(refKey, entityId),
			).
			Commit()

		if txnErr != nil {
			log.Warn("failed to write short-id migration", "entity", entityId, "error", txnErr)
			errorCount++
			continue
		}

		if !txnResp.Succeeded {
			// Ref was claimed by a concurrent operation; try again with a new id
			log.Warn("short-id collision during migration, skipping", "entity", entityId, "short_id", shortId)
			errorCount++
			continue
		}

		existingRefs[shortId] = struct{}{}
		migrated++

		if migrated%100 == 0 {
			log.Info("short-id migration progress", "migrated", migrated, "skipped", skipped)
		}
	}

	if errorCount > 0 {
		log.Warn("short-id migration completed with errors",
			"migrated", migrated, "skipped", skipped, "errors", errorCount)
		return migrated, skipped, fmt.Errorf("short-id migration completed with %d errors", errorCount)
	}

	log.Info("short-id migration completed", "migrated", migrated, "skipped", skipped)
	return migrated, skipped, nil
}
