//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	webResearchMCPAlias    = "webresearchmcp"
	webResearchMCPPort     = "8095/tcp"
	envWebResearchMCPImage = "WEBRESEARCHMCP_IMAGE"
	toolEndpointPath       = "/mcp"
	pageFetchWait          = "90s"
)

func startWebResearchMCP(t *testing.T, ctx context.Context, networkName string) string {
	t.Helper()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          webResearchMCPImage(t),
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {webResearchMCPAlias}},
			ExposedPorts:   []string{webResearchMCPPort},
			WaitingFor:     wait.ForListeningPort(webResearchMCPPort),
			Env: map[string]string{
				"SEARXNG_URL":             searxngNetworkURL(),
				"SCRAPE_REQUEST_NATS_URL": natsjetstream.NetworkURL(),
				"PAGE_MARKDOWN_NATS_URL":  natsjetstream.NetworkURL(),
				"CORPUSMARKDOWN_ADDR":     corpusMarkdownNetworkAddress(),
				"PAGE_FETCH_WAIT":         pageFetchWait,
				"LOG_LEVEL":               "debug",
			},
		},
	})
	if err != nil {
		t.Fatalf("start webresearchmcp container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "webresearchmcp", container)

	return containerurl.HostURL(t, ctx, container, webResearchMCPPort) + toolEndpointPath
}

func webResearchMCPImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envWebResearchMCPImage, "webresearchmcp", "e2e")
}
