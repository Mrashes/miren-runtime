package rabbitmq

import (
	"miren.dev/runtime/pkg/addon"
)

const (
	AddonName    = "miren-rabbitmq"
	DefaultImage = "docker.io/library/rabbitmq:4"
)

func Definition() addon.AddonDefinition {
	return addon.AddonDefinition{
		Name:           AddonName,
		DisplayName:    "Miren RabbitMQ",
		Description:    "Managed RabbitMQ message broker",
		DefaultVariant: "small",
		Variants: []addon.VariantDefinition{
			{
				Name:        "small",
				Description: "Dedicated RabbitMQ server",
			},
		},
	}
}
