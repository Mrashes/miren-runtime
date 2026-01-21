package ingress

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/ingress/ingress_v1alpha"
	"miren.dev/runtime/components/autotls"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/rpc/stream"
)

// RouteWatcher watches http_route entities and maintains an in-memory set
// of hosts with configured routes. It implements autotls.RouteWatcher.
type RouteWatcher struct {
	log   *slog.Logger
	eac   *entityserver_v1alpha.EntityAccessClient
	hosts *autotls.RouteSet
}

// NewRouteWatcher creates a new RouteWatcher.
func NewRouteWatcher(log *slog.Logger, eac *entityserver_v1alpha.EntityAccessClient) *RouteWatcher {
	return &RouteWatcher{
		log:   log.With("module", "route-watcher"),
		eac:   eac,
		hosts: autotls.NewRouteSet(),
	}
}

// RouteSet returns the underlying RouteSet for use with autotls.ServeTLS.
func (rw *RouteWatcher) RouteSet() *autotls.RouteSet {
	return rw.hosts
}

// Start loads all existing routes and begins watching for changes.
// It blocks until the context is cancelled.
func (rw *RouteWatcher) Start(ctx context.Context) error {
	// Load existing routes
	if err := rw.loadExistingRoutes(ctx); err != nil {
		return err
	}

	rw.log.Info("starting route watcher")

	// Watch for changes
	index := entity.Ref(entity.EntityKind, ingress_v1alpha.KindHttpRoute)

	_, err := rw.eac.WatchIndex(ctx, index, stream.Callback(func(op *entityserver_v1alpha.EntityOp) error {
		if op == nil {
			return nil
		}

		var route ingress_v1alpha.HttpRoute
		route.Decode(op.Entity().Entity())

		host := strings.ToLower(strings.TrimSpace(route.Host))
		if host == "" {
			// Default routes have no host - skip them for cert provisioning purposes
			return nil
		}

		switch op.OperationType() {
		case entityserver_v1alpha.EntityOperationCreate:
			rw.log.Debug("route created", "host", host)
			rw.hosts.Add(host)
		case entityserver_v1alpha.EntityOperationUpdate:
			// For updates, we just make sure the host is in the set
			// (host changes on a route are unlikely but handled)
			rw.hosts.Add(host)
		case entityserver_v1alpha.EntityOperationDelete:
			rw.log.Debug("route deleted", "host", host)
			rw.hosts.Remove(host)
		}

		return nil
	}))

	return err
}

// loadExistingRoutes loads all existing http_route entities into the set.
func (rw *RouteWatcher) loadExistingRoutes(ctx context.Context) error {
	res, err := rw.eac.List(ctx, entity.Ref(entity.EntityKind, ingress_v1alpha.KindHttpRoute))
	if err != nil {
		return err
	}

	var hosts []string
	for _, ent := range res.Values() {
		var route ingress_v1alpha.HttpRoute
		route.Decode(ent.Entity())

		host := strings.ToLower(strings.TrimSpace(route.Host))
		if host != "" {
			hosts = append(hosts, host)
		}
	}

	rw.hosts.Replace(hosts)
	rw.log.Info("loaded existing routes", "count", len(hosts))

	return nil
}

// StartBackground starts the watcher in a goroutine and returns immediately.
// Use this when you need the watcher running but don't want to block.
func (rw *RouteWatcher) StartBackground(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := rw.Start(ctx); err != nil {
			rw.log.Error("route watcher error", "error", err)
		}
	}()
}
