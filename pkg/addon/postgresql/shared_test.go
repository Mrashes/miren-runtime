package postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"miren.dev/runtime/pkg/addon"
	"miren.dev/runtime/pkg/saga"
)

func TestRegisterSharedSaga(t *testing.T) {
	registry := saga.NewRegistry()
	fw := &addon.ProviderFramework{}
	rc := &resultCapture{}

	err := RegisterSharedSaga(registry, fw, rc)
	require.NoError(t, err)

	def, ok := registry.Get("provision-shared-postgresql")
	require.True(t, ok)
	assert.Equal(t, "provision-shared-postgresql", def.Name)
	assert.Len(t, def.Actions, 6)
}

func TestRegisterDeprovisionSharedSaga(t *testing.T) {
	registry := saga.NewRegistry()
	fw := &addon.ProviderFramework{}

	err := RegisterDeprovisionSharedSaga(registry, fw)
	require.NoError(t, err)

	def, ok := registry.Get("deprovision-shared-postgresql")
	require.True(t, ok)
	assert.Equal(t, "deprovision-shared-postgresql", def.Name)
	assert.Len(t, def.Actions, 7)
}

func TestSharedSagaActionOrder(t *testing.T) {
	registry := saga.NewRegistry()
	fw := &addon.ProviderFramework{}
	rc := &resultCapture{}

	err := RegisterSharedSaga(registry, fw, rc)
	require.NoError(t, err)

	def, ok := registry.Get("provision-shared-postgresql")
	require.True(t, ok)

	expectedActions := []string{
		"find-or-create-shared-server",
		"generate-shared-credentials",
		"create-shared-user",
		"create-shared-database",
		"increment-association-count",
		"build-shared-result",
	}

	for _, name := range expectedActions {
		_, exists := def.Actions[name]
		assert.True(t, exists, "expected action %q to exist", name)
	}
}

func TestDeprovisionSharedSagaActions(t *testing.T) {
	registry := saga.NewRegistry()
	fw := &addon.ProviderFramework{}

	err := RegisterDeprovisionSharedSaga(registry, fw)
	require.NoError(t, err)

	def, ok := registry.Get("deprovision-shared-postgresql")
	require.True(t, ok)

	expectedActions := []string{
		"decode-shared-attrs",
		"lookup-shared-server",
		"terminate-connections",
		"drop-shared-database",
		"drop-shared-user",
		"decrement-association-count",
		"cleanup-shared-server",
	}

	for _, name := range expectedActions {
		_, exists := def.Actions[name]
		assert.True(t, exists, "expected action %q to exist", name)
	}
}
