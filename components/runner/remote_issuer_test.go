package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"miren.dev/runtime/api/runner/runner_v1alpha"
	"miren.dev/runtime/pkg/caauth"
	"miren.dev/runtime/pkg/entity/testutils"
	"miren.dev/runtime/pkg/rpc"
	"miren.dev/runtime/pkg/workloadidentity"
	runnersrv "miren.dev/runtime/servers/runner"
)

// newTestRegistrationClient builds a RunnerRegistration client backed by a real
// in-process RegistrationServer, optionally with a workload identity issuer.
func newTestRegistrationClient(t *testing.T, issuer *workloadidentity.Issuer) *runner_v1alpha.RunnerRegistrationClient {
	t.Helper()

	es, cleanup := testutils.NewInMemEntityServer(t)
	t.Cleanup(cleanup)

	ca, err := caauth.New(caauth.Options{CommonName: "test-ca", Organization: "test"})
	require.NoError(t, err)

	srv := runnersrv.NewRegistrationServer(runnersrv.RegistrationServerConfig{
		Log:            testutils.TestLogger(t),
		Authority:      ca,
		EAC:            es.EAC,
		WorkloadIssuer: issuer,
	})

	local := rpc.LocalClient(runner_v1alpha.AdaptRunnerRegistration(srv))
	return runner_v1alpha.NewRunnerRegistrationClient(local)
}

func TestRemoteIssuerRefreshesURL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	issuer, err := workloadidentity.NewIssuer(workloadidentity.IssuerConfig{
		DataPath:       t.TempDir(),
		IssuerURL:      "https://updated.example",
		OrganizationID: "org-test",
		ClusterID:      "cluster-test",
	})
	require.NoError(t, err)

	client := newTestRegistrationClient(t, issuer)

	ri := newRemoteIssuer(ctx, testutils.TestLogger(t), client, "https://stale.example")
	require.Equal(t, "https://stale.example", ri.IssuerURL())

	ri.refreshIssuerURL()
	require.Equal(t, "https://updated.example", ri.IssuerURL())
}

func TestRemoteIssuerSetEnabledTransitions(t *testing.T) {
	// setEnabled reports true only when the state actually changes, so the
	// refresh loop logs a transition once rather than on every interval.
	ri := &remoteIssuer{enabled: true}
	require.False(t, ri.setEnabled(true), "no change should report false")
	require.True(t, ri.setEnabled(false), "enabled->disabled should report true")
	require.False(t, ri.setEnabled(false), "repeated disabled should report false")
	require.True(t, ri.setEnabled(true), "disabled->enabled should report true")
}

func TestRemoteIssuerRefreshKeepsURLWhenDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// No issuer configured on the coordinator -> WorkloadIssuerInfo reports
	// disabled. The cached URL must be preserved rather than cleared.
	client := newTestRegistrationClient(t, nil)

	ri := newRemoteIssuer(ctx, testutils.TestLogger(t), client, "https://stale.example")
	ri.refreshIssuerURL()
	require.Equal(t, "https://stale.example", ri.IssuerURL())
}
