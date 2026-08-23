//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	renderproxyAlias    = "renderproxy"
	renderproxyPort     = "8080"
	envRenderproxyImage = "RENDERPROXY_IMAGE"
)

func startRenderproxy(
	t *testing.T,
	ctx context.Context,
	networkName string,
	cdpURL string,
	extraEnv map[string]string,
) string {
	t.Helper()
	egressproxy.Start(t, ctx, networkName)
	env := map[string]string{
		"RENDERPROXY_CDP_URL": cdpURL,
		"EGRESS_PROXY_URL":    egressproxy.NetworkURL(),
		"LOG_LEVEL":           "debug",
	}
	for k, v := range extraEnv {
		env[k] = v
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          renderproxyImage(t),
			ExposedPorts:   []string{renderproxyPort + "/tcp"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {renderproxyAlias}},
			Env:            env,
			WaitingFor:     wait.ForListeningPort(renderproxyPort + "/tcp"),
		},
	})
	if err != nil {
		t.Fatalf("start renderproxy container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, renderproxyAlias, container)

	return containerurl.HostURL(t, ctx, container, renderproxyPort+"/tcp")
}

func renderproxyImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envRenderproxyImage, "renderproxy", "e2e")
}
