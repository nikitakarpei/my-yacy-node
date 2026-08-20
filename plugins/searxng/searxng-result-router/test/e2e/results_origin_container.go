//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/searxng"
)

const (
	resultsOriginImage    = "docker.io/library/caddy:2.11.4-alpine"
	resultsOriginPort     = "8080/tcp"
	resultsOriginFilePath = "/etc/caddy/Caddyfile"
	visitPath             = "/visit"
)

func startResultsOrigin(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        resultsOriginImage,
			ExposedPorts: []string{resultsOriginPort},
			Networks:     []string{networkName},
			Files: []testcontainers.ContainerFile{{
				Reader:            strings.NewReader(resultsOriginRouting()),
				ContainerFilePath: resultsOriginFilePath,
				FileMode:          0o644,
			}},
			WaitingFor: wait.ForHTTP("/").
				WithPort(resultsOriginPort).
				WithStartupTimeout(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("start results origin container %s: %v", resultsOriginImage, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "results-origin", container)
	return containerurl.HostURL(t, ctx, container, resultsOriginPort)
}

func resultsOriginRouting() string {
	return fmt.Sprintf(`:%s {
	handle %s* {
		reverse_proxy %s
	}
	handle {
		reverse_proxy %s
	}
}
`,
		portNumberOf(resultsOriginPort),
		visitPath,
		visitcrawlAlias+":"+portNumberOf(visitcrawlPort),
		searxngAlias+":"+portNumberOf(searxng.Port),
	)
}

func portNumberOf(containerPort string) string {
	return strings.TrimSuffix(containerPort, "/tcp")
}
