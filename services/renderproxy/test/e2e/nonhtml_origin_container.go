//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
)

const (
	nonhtmlImage       = "docker.io/library/nginx:alpine"
	nonhtmlPath        = "/payload.json"
	nonhtmlContentType = "application/json"
	nonhtmlPayload     = `{"marker":"raw-body-marker"}`
)

func startNonHTMLOrigin(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          nonhtmlImage,
			ExposedPorts:   []string{"80/tcp"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {originAlias}},
			Files: []testcontainers.ContainerFile{{
				Reader:            strings.NewReader(nonhtmlPayload),
				ContainerFilePath: "/usr/share/nginx/html" + nonhtmlPath,
				FileMode:          0o644,
			}},
			WaitingFor: wait.ForHTTP(nonhtmlPath).WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start non-html origin container %s: %v", nonhtmlImage, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, originAlias, container)
	return "http://" + originAlias + nonhtmlPath
}
