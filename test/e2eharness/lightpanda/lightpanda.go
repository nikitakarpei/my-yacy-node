//go:build e2e

package lightpanda

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
)

const (
	image       = "lightpanda/browser:0.3.4"
	alias       = "lightpanda"
	port        = "9222"
	pathVersion = "/json/version"
)

func Start(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	start(t, ctx, networkName, nil)
}

// StartWithDefaultProxy starts the browser with a proxy of its own, which it uses for a
// page whose caller states no proxy.
func StartWithDefaultProxy(
	t *testing.T,
	ctx context.Context,
	networkName string,
	defaultProxyURL string,
) {
	t.Helper()
	start(t, ctx, networkName, []string{"--http-proxy", defaultProxyURL})
}

func start(t *testing.T, ctx context.Context, networkName string, extraArgs []string) {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          image,
			ExposedPorts:   []string{port + "/tcp"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			Cmd: append([]string{
				"/bin/lightpanda", "serve",
				"--host", "0.0.0.0",
				"--port", port,
				"--advertise-host", alias,
			}, extraArgs...),
			WaitingFor: wait.ForHTTP(pathVersion).
				WithPort(port + "/tcp").
				WithForcedIPv4LocalHost().
				WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start lightpanda container %s: %v", image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, alias, container)
}

func NetworkURL() string {
	return "http://" + alias + ":" + port
}
