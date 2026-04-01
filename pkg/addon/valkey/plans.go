package valkey

import (
	"miren.dev/runtime/pkg/addon"
)

const (
	AddonName    = "miren-valkey"
	DefaultImage = "docker.io/valkey/valkey:8"
)

const (
	ConfigStorage = "storage"
)

func Definition() addon.AddonDefinition {
	return addon.AddonDefinition{
		Name:           AddonName,
		DisplayName:    "Miren Valkey",
		Description:    "Managed Valkey key-value store",
		DefaultVariant: "small",
		Variants: []addon.VariantDefinition{
			{
				Name:        "small",
				Description: "Dedicated Valkey server",
				Details: map[string]string{
					"Storage": "1 GB",
				},
				Config: map[string]string{
					ConfigStorage: "1Gi",
				},
			},
		},
	}
}
