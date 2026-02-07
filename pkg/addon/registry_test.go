package addon

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDefinition() AddonDefinition {
	return AddonDefinition{
		Name:        "test-addon",
		DisplayName: "Test Addon",
		Description: "A test addon",
		DefaultPlan: "small",
		Plans: []PlanDefinition{
			{
				Name:        "small",
				Description: "Small plan",
				Details:     map[string]string{"CPU": "0.5"},
				Config:      map[string]string{"cpu": "500m"},
			},
			{
				Name:        "large",
				Description: "Large plan",
				Details:     map[string]string{"CPU": "2"},
				Config:      map[string]string{"cpu": "2000m"},
			},
		},
	}
}

type mockProvider struct{}

func (m *mockProvider) Provision(ctx context.Context, app App, plan Plan) (*ProvisionResult, error) {
	return &ProvisionResult{}, nil
}
func (m *mockProvider) AdjustEnvVars(ctx context.Context, result *ProvisionResult, assoc AddonAssociation, collisions []string) ([]Variable, error) {
	return nil, nil
}
func (m *mockProvider) Deprovision(ctx context.Context, assoc AddonAssociation) error {
	return nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	def := testDefinition()
	provider := &mockProvider{}

	r.Register("test-addon", provider, def)

	p, d, ok := r.Get("test-addon")
	require.True(t, ok)
	assert.Equal(t, def.Name, d.Name)
	assert.Equal(t, def.DisplayName, d.DisplayName)
	assert.NotNil(t, p)
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry()

	_, _, ok := r.Get("nonexistent")
	assert.False(t, ok)
}

func TestRegistryListAddons(t *testing.T) {
	r := NewRegistry()
	r.Register("addon-a", &mockProvider{}, AddonDefinition{Name: "addon-a"})
	r.Register("addon-b", &mockProvider{}, AddonDefinition{Name: "addon-b"})

	defs := r.ListAddons()
	assert.Len(t, defs, 2)

	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}
	assert.True(t, names["addon-a"])
	assert.True(t, names["addon-b"])
}

func TestResolveAddonAndPlanExplicit(t *testing.T) {
	r := NewRegistry()
	r.Register("test-addon", &mockProvider{}, testDefinition())

	name, plan, err := r.ResolveAddonAndPlan("test-addon:large")
	require.NoError(t, err)
	assert.Equal(t, "test-addon", name)
	assert.Equal(t, "large", plan)
}

func TestResolveAddonAndPlanDefault(t *testing.T) {
	r := NewRegistry()
	r.Register("test-addon", &mockProvider{}, testDefinition())

	name, plan, err := r.ResolveAddonAndPlan("test-addon")
	require.NoError(t, err)
	assert.Equal(t, "test-addon", name)
	assert.Equal(t, "small", plan) // default plan
}

func TestResolveAddonAndPlanUnknownAddon(t *testing.T) {
	r := NewRegistry()

	_, _, err := r.ResolveAddonAndPlan("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown addon")
}

func TestResolveAddonAndPlanUnknownPlan(t *testing.T) {
	r := NewRegistry()
	r.Register("test-addon", &mockProvider{}, testDefinition())

	_, _, err := r.ResolveAddonAndPlan("test-addon:nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown plan")
}

func TestGetPlanConfig(t *testing.T) {
	r := NewRegistry()
	r.Register("test-addon", &mockProvider{}, testDefinition())

	config, err := r.GetPlanConfig("test-addon", "small")
	require.NoError(t, err)
	assert.Equal(t, "500m", config["cpu"])

	config, err = r.GetPlanConfig("test-addon", "large")
	require.NoError(t, err)
	assert.Equal(t, "2000m", config["cpu"])
}

func TestGetPlanConfigUnknownAddon(t *testing.T) {
	r := NewRegistry()

	_, err := r.GetPlanConfig("unknown", "small")
	assert.Error(t, err)
}

func TestGetPlanConfigUnknownPlan(t *testing.T) {
	r := NewRegistry()
	r.Register("test-addon", &mockProvider{}, testDefinition())

	_, err := r.GetPlanConfig("test-addon", "nonexistent")
	assert.Error(t, err)
}
