package rabbitmq

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"miren.dev/runtime/pkg/addon"
	"miren.dev/runtime/pkg/addon/dbsaga"
)

type Provider struct {
	dbsaga.BaseProvider
}

func NewProvider(fw *addon.ProviderFramework) *Provider {
	return &Provider{
		BaseProvider: dbsaga.BaseProvider{
			Fw:  fw,
			Log: fw.Log.With("addon", AddonName),
		},
	}
}

func (p *Provider) Provision(ctx context.Context, app addon.App, variant addon.Variant) (*addon.ProvisionResult, error) {
	return p.provisionDedicated(ctx, app, variant)
}

func (p *Provider) Deprovision(ctx context.Context, assoc addon.AddonAssociation) error {
	return p.deprovisionDedicated(ctx, assoc)
}

func buildRabbitmqURL(user, password, host string, port int, vhost string) string {
	u := url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, fmt.Sprintf("%d", port)),
		Path:   "/" + vhost,
	}
	return u.String()
}

func buildEnvVars(user, password, host string, port int, vhost string) []addon.Variable {
	portStr := fmt.Sprintf("%d", port)

	return []addon.Variable{
		{Key: "RABBITMQ_URL", Value: buildRabbitmqURL(user, password, host, port, vhost), Sensitive: true},
		{Key: "RABBITMQ_HOST", Value: host},
		{Key: "RABBITMQ_PORT", Value: portStr},
		{Key: "RABBITMQ_USER", Value: user},
		{Key: "RABBITMQ_PASSWORD", Value: password, Sensitive: true},
		{Key: "RABBITMQ_VHOST", Value: vhost},
	}
}
