package postgresql

import "miren.dev/runtime/pkg/addon"

const (
	AddonName    = "miren-postgresql"
	DefaultImage = "postgres:17"
)

// Plan configuration keys
const (
	ConfigCPU     = "cpu"
	ConfigMemory  = "memory"
	ConfigStorage = "storage"
	ConfigShared  = "shared"
)

// Definition returns the addon definition for PostgreSQL.
func Definition() addon.AddonDefinition {
	return addon.AddonDefinition{
		Name:        AddonName,
		DisplayName: "Miren PostgreSQL",
		Description: "Managed PostgreSQL database",
		DefaultPlan: "small-local",
		Plans: []addon.PlanDefinition{
			{
				Name:        "small-local",
				Description: "Development and testing",
				Details: map[string]string{
					"CPU":     "0.5 cores",
					"Memory":  "512 MB",
					"Storage": "1 GB",
				},
				Config: map[string]string{
					ConfigCPU:     "500m",
					ConfigMemory:  "512Mi",
					ConfigStorage: "1Gi",
					ConfigShared:  "false",
				},
			},
			{
				Name:        "medium-local",
				Description: "Small production workloads",
				Details: map[string]string{
					"CPU":     "1 core",
					"Memory":  "1 GB",
					"Storage": "10 GB",
				},
				Config: map[string]string{
					ConfigCPU:     "1000m",
					ConfigMemory:  "1Gi",
					ConfigStorage: "10Gi",
					ConfigShared:  "false",
				},
			},
			{
				Name:        "large-local",
				Description: "Production workloads",
				Details: map[string]string{
					"CPU":     "2 cores",
					"Memory":  "4 GB",
					"Storage": "50 GB",
				},
				Config: map[string]string{
					ConfigCPU:     "2000m",
					ConfigMemory:  "4Gi",
					ConfigStorage: "50Gi",
					ConfigShared:  "false",
				},
			},
			{
				Name:        "shared",
				Description: "Multi-app shared server (cost-effective)",
				Details: map[string]string{
					"Type": "Shared server",
					"Note": "Multiple apps share one PostgreSQL instance",
				},
				Config: map[string]string{
					ConfigShared: "true",
				},
			},
		},
	}
}

// IsSharedPlan returns true if the plan is a shared-server plan.
func IsSharedPlan(planName string) bool {
	return planName == "shared"
}
