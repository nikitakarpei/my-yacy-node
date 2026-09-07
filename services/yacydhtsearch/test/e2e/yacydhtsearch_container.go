//go:build e2e

package e2e

import (
	"context"
	"maps"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
)

const (
	yacydhtsearchAlias    = "yacydhtsearch"
	yacydhtsearchPort     = "8080"
	yacydhtsearchOpsPort  = "9090"
	envYacydhtsearchImage = "YACYDHTSEARCH_IMAGE"
)

type yacydhtsearchService struct {
	searchURL string
	opsURL    string
}

func startYacydhtsearch(
	t *testing.T,
	ctx context.Context,
	networkName string,
	alias string,
	seedlistURL string,
	settings map[string]string,
) yacydhtsearchService {
	t.Helper()
	environment := map[string]string{
		"YACYDHTSEARCH_SEEDLIST_URLS": seedlistURL,
		"EGRESS_PROXY_URL":            egressproxy.NetworkURL(),
		"LOG_LEVEL":                   "debug",
	}
	maps.Copy(environment, settings)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          yacydhtsearchImage(t),
			ExposedPorts:   []string{yacydhtsearchPort + "/tcp", yacydhtsearchOpsPort + "/tcp"},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			Env:            environment,
			WaitingFor:     wait.ForListeningPort(yacydhtsearchPort + "/tcp"),
		},
	})
	if err != nil {
		t.Fatalf("start yacydhtsearch container %s: %v", alias, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, alias, container)

	return yacydhtsearchService{
		searchURL: containerurl.HostURL(t, ctx, container, yacydhtsearchPort+"/tcp"),
		opsURL:    containerurl.HostURL(t, ctx, container, yacydhtsearchOpsPort+"/tcp"),
	}
}

func yacydhtsearchImage(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envYacydhtsearchImage, "yacydhtsearch", "e2e")
}
