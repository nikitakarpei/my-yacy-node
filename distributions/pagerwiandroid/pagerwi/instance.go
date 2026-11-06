// Package pagerwi runs the pagerwi service inside an Android app. The app binds this
// package with gomobile and gives it the environment variables of the service as text.
package pagerwi

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/environmenttext"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehost"
)

type Instance struct {
	host *servicehost.Host
}

func InstanceFrom(environmentText string) (*Instance, error) {
	variables, err := environmenttext.VariablesFrom(environmentText)
	if err != nil {
		return nil, err
	}
	if err := applog.Configure(variables.Get); err != nil {
		return nil, err
	}
	config, err := pagerwiassembly.LoadServiceConfig(variables.Get)
	if err != nil {
		return nil, err
	}
	return &Instance{host: servicehost.HostFor(config.OpsAddr, serviceFor(config))}, nil
}

func serviceFor(config pagerwiassembly.ServiceConfig) servicehost.Service {
	return func(ctx context.Context, registry *prometheus.Registry) error {
		listeners, err := pagerwiassembly.ListenersFor(ctx, config)
		if err != nil {
			return err
		}
		return pagerwiassembly.RunService(ctx, config, listeners, registry)
	}
}

func (i *Instance) Run() error {
	return i.host.Run()
}

func (i *Instance) Stop() {
	i.host.Stop()
}
