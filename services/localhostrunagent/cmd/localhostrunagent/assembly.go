package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/internal/localhostruntunnel"
	"github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/internal/proxyprotocolingress"
	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/processenvironmentleaseserver"
)

const (
	advertiseHostName = "YACY_ADVERTISE_HOST"
	advertisePortName = "YACY_ADVERTISE_PORT"
)

func RunAgent(ctx context.Context, configuration AgentConfiguration) error {
	lease, err := processenvironmentleaseserver.Listen(ctx, configuration.LeaseSocketPath)
	if err != nil {
		return err
	}
	defer closed(ctx, "lease socket", lease)

	ingress, err := proxyprotocolingress.Listen(ctx, proxyprotocolingress.Configuration{
		Origin:        configuration.NodeOrigin,
		EgressAddress: configuration.AgentAddress,
	})
	if err != nil {
		return err
	}
	defer closed(ctx, "private ingress", ingress)

	tunnel, err := localhostruntunnel.Open(ctx, localhostruntunnel.Configuration{
		Host:             configuration.LocalhostRunHost,
		IdentityFilePath: configuration.IdentityFilePath,
		KnownHostsPath:   configuration.KnownHostsPath,
		IngressPort:      ingress.Port(),
	})
	if err != nil {
		return err
	}
	defer closed(ctx, "tunnel", tunnel)

	lease.Grant(grantOf(tunnel.Hostname()))

	slog.InfoContext(ctx, "localhostrunagent started",
		slog.String("hostname", tunnel.Hostname()),
	)

	return firstFailureOf(ctx, lease, ingress, tunnel)
}

func grantOf(hostname string) processenvironmentlease.Grant {
	return processenvironmentlease.Grant{
		ProcessEnvironment: map[string]string{
			advertiseHostName: hostname,
			advertisePortName: strconv.Itoa(localhostruntunnel.PublicPort),
		},
	}
}

func firstFailureOf(
	ctx context.Context,
	lease *processenvironmentleaseserver.Server,
	ingress *proxyprotocolingress.Ingress,
	tunnel *localhostruntunnel.Tunnel,
) error {
	select {
	case <-ctx.Done():
		slog.InfoContext(ctx, "localhostrunagent stopping")

		return nil
	case err := <-lease.Failure():
		return fmt.Errorf("hold the process environment lease: %w", err)
	case err := <-ingress.Failure():
		return fmt.Errorf("hold the private ingress: %w", err)
	case err := <-tunnel.Failure():
		return fmt.Errorf("hold the tunnel: %w", err)
	}
}

func closed(ctx context.Context, unit string, closer io.Closer) {
	if err := closer.Close(); err != nil {
		slog.WarnContext(ctx, "unit not closed",
			slog.String("unit", unit),
			slog.Any("error", err),
		)
	}
}
