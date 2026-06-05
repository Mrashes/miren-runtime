package runner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"miren.dev/runtime/api/runner/runner_v1alpha"
	"miren.dev/runtime/pkg/workloadidentity"
)

const (
	// remoteTokenTimeout bounds a single token-minting RPC to the coordinator.
	remoteTokenTimeout = 30 * time.Second

	// issuerURLRefreshInterval is how often the cached issuer URL is re-synced
	// with the coordinator.
	issuerURLRefreshInterval = 5 * time.Minute
)

// remoteIssuer satisfies workloadidentity.TokenIssuer by proxying token minting
// to the coordinator over RPC. Distributed runners do not hold the cluster
// signing key, so they cannot mint tokens locally and instead ask the
// coordinator, which holds the key.
//
// The issuer URL is cached but kept in sync by a background loop: it can change
// while the runner is running (e.g. the cluster gains a DNS hostname during
// re-registration), and the RPC client transparently reconnects after a
// coordinator restart, so periodic polling re-syncs without a runner restart.
type remoteIssuer struct {
	ctx    context.Context
	client *runner_v1alpha.RunnerRegistrationClient
	log    *slog.Logger

	mu        sync.RWMutex
	issuerURL string
	enabled   bool
}

var _ workloadidentity.TokenIssuer = (*remoteIssuer)(nil)

func newRemoteIssuer(ctx context.Context, log *slog.Logger, client *runner_v1alpha.RunnerRegistrationClient, issuerURL string) *remoteIssuer {
	r := &remoteIssuer{
		ctx:       ctx,
		client:    client,
		log:       log,
		issuerURL: issuerURL,
		// A remoteIssuer is only constructed once the coordinator has reported
		// an enabled issuer, so start in the enabled state.
		enabled: true,
	}
	go r.refreshLoop()
	return r
}

func (r *remoteIssuer) IssuerURL() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.issuerURL
}

func (r *remoteIssuer) setIssuerURL(url string) {
	r.mu.Lock()
	r.issuerURL = url
	r.mu.Unlock()
}

// setEnabled records the latest enabled state and reports whether it changed,
// so transitions can be logged once instead of on every refresh.
func (r *remoteIssuer) setEnabled(enabled bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.enabled == enabled {
		return false
	}
	r.enabled = enabled
	return true
}

func (r *remoteIssuer) IssueToken(app, sandboxID string) (string, error) {
	return r.IssueTokenWithOptions(app, sandboxID, workloadidentity.TokenOptions{})
}

// IssueTokenWithOptions mints a token via the coordinator. The app argument is
// ignored: the coordinator derives the app identity from the sandbox itself so
// a runner cannot forge it.
func (r *remoteIssuer) IssueTokenWithOptions(_, sandboxID string, opts workloadidentity.TokenOptions) (string, error) {
	ctx, cancel := context.WithTimeout(r.ctx, remoteTokenTimeout)
	defer cancel()

	var ttlSeconds int64
	if opts.TTL > 0 {
		ttlSeconds = int64(opts.TTL / time.Second)
	}

	res, err := r.client.IssueWorkloadToken(ctx, sandboxID, opts.Audience, ttlSeconds)
	if err != nil {
		return "", fmt.Errorf("requesting workload token from coordinator: %w", err)
	}
	if res.Error() != "" {
		return "", fmt.Errorf("coordinator refused workload token: %s", res.Error())
	}
	return res.Token(), nil
}

// refreshLoop keeps the cached issuer URL in sync with the coordinator until the
// runner's context is cancelled.
func (r *remoteIssuer) refreshLoop() {
	ticker := time.NewTicker(issuerURLRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.refreshIssuerURL()
		}
	}
}

func (r *remoteIssuer) refreshIssuerURL() {
	ctx, cancel := context.WithTimeout(r.ctx, remoteTokenTimeout)
	defer cancel()

	info, err := r.client.WorkloadIssuerInfo(ctx)
	if err != nil {
		r.log.Warn("failed to refresh workload issuer info", "error", err)
		return
	}

	enabled := info.Enabled()
	if r.setEnabled(enabled) {
		// Log only on transitions to avoid spamming every refresh interval.
		if enabled {
			r.log.Info("coordinator re-enabled workload identity issuer")
		} else {
			r.log.Warn("coordinator disabled workload identity issuer; sandbox token issuance will fail until re-enabled")
		}
	}
	if !enabled {
		// Leave the cached URL as-is so already-injected sandbox env values
		// stay coherent.
		return
	}
	if url := info.IssuerUrl(); url != "" && url != r.IssuerURL() {
		r.log.Info("workload issuer URL updated", "issuer", url)
		r.setIssuerURL(url)
	}
}
