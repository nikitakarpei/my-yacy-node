//go:build e2e

// Package yacypeer starts, restarts, and feeds documents into the real YaCy
// peer.
package yacypeer

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerclient"
)

const defaultImage = "docker.io/yacy/yacy_search_server:latest"

func Start(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	networkName, alias string,
	configOverrides ...string,
) (testcontainers.Container, string) {
	t.Helper()
	image := os.Getenv("YACY_YACY_IMAGE")
	if image == "" {
		image = defaultImage
	}
	const defaults = "/opt/yacy_search_server/defaults/"
	const unitFile = defaults + "yacy.network.freeworld.unit"
	setup := strings.Join(append([]string{
		"sed -i 's#<auth-method>DIGEST</auth-method>#<auth-method>BASIC</auth-method>#' " + defaults + "web.xml",
		"sed -i '/^network.unit.bootstrap.seedlist/d' " + unitFile,
		"sed -i 's#^network.unit.domain.*#network.unit.domain = any#' " + unitFile,
		"sed -i 's#^staticIP=.*#staticIP=" + alias + "#' " + defaults + "yacy.init",
		"sed -i 's#^allowDistributeIndex=.*#allowDistributeIndex=true#' " + defaults + "yacy.init",
		"sed -i 's#^allowDistributeIndexWhileCrawling=.*#allowDistributeIndexWhileCrawling=true#' " + defaults + "yacy.init",
		"sed -i 's#^allowDistributeIndexWhileIndexing=.*#allowDistributeIndexWhileIndexing=true#' " + defaults + "yacy.init",
		"sed -i 's#^20_dhtdistribution_loadprereq=.*#20_dhtdistribution_loadprereq=999.0#' " + defaults + "yacy.init",
		"sed -i 's#^20_dhtreceive_loadprereq=.*#20_dhtreceive_loadprereq=999.0#' " + defaults + "yacy.init",
		"sed -i 's#^30_peerping_loadprereq=.*#30_peerping_loadprereq=999.0#' " + defaults + "yacy.init",
		"sed -i 's#^30_peerping_busysleep=.*#30_peerping_busysleep=10000#' " + defaults + "yacy.init",
		"sed -i 's#^20_dhtdistribution_idlesleep=.*#20_dhtdistribution_idlesleep=1000#' " + defaults + "yacy.init",
		"sed -i 's#^20_dhtdistribution_busysleep=.*#20_dhtdistribution_busysleep=0#' " + defaults + "yacy.init",
		"sed -i 's#^.level=.*#.level=FINE#' " + defaults + "yacy.logging",
		"sed -i 's#^NETWORK.level.*#NETWORK.level = FINE#' " + defaults + "yacy.logging",
		"sed -i 's#^SWITCHBOARD.level.*#SWITCHBOARD.level = FINE#' " + defaults + "yacy.logging",
		"sed -i 's#^DHT-OUT.level.*#DHT-OUT.level = FINE#' " + defaults + "yacy.logging",
	}, configOverrides...), " && ")
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          image,
			ExposedPorts:   []string{peerclient.ExposedPort},
			Networks:       []string{networkName},
			NetworkAliases: map[string][]string{networkName: {alias}},
			WaitingFor:     wait.ForExec([]string{"true"}).WithStartupTimeout(2 * time.Minute),
			Cmd: []string{
				"/bin/sh", "-c",
				setup + " && exec /bin/sh /opt/yacy_search_server/startYACY.sh -f",
			},
		},
	})
	if err != nil {
		t.Fatalf("start YaCy container %s: %v", image, err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "yacy", container)
	yacyURL := containerurl.HostURL(t, ctx, container, peerclient.ExposedPort)
	if !pollwait.For(60*time.Second, func() bool {
		return probe.OK(ctx, yacyURL+"/yacy/query.html?object=rwicount")
	}) {
		t.Fatal("YaCy never became reachable from the host")
	}
	return container, yacyURL
}

func Restart(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	container testcontainers.Container,
) string {
	t.Helper()
	stopTimeout := 60 * time.Second
	if err := container.Stop(ctx, &stopTimeout); err != nil {
		t.Fatalf("stop yacy: %v", err)
	}
	requireFlushedShutdown(t, ctx, container)
	if err := container.Start(ctx); err != nil {
		t.Fatalf("restart yacy: %v", err)
	}
	yacyURL := containerurl.HostURL(t, ctx, container, peerclient.ExposedPort)
	if !pollwait.For(60*time.Second, func() bool {
		return probe.OK(ctx, yacyURL+"/yacy/query.html?object=rwicount")
	}) {
		t.Fatal("YaCy never became reachable after restart")
	}
	return yacyURL
}

const killedOnStopExitCode = 137

// requireFlushedShutdown fails when the stop timeout expired and YaCy was
// killed, because the peer then restarts with none of the RWIs it held.
func requireFlushedShutdown(t *testing.T, ctx context.Context, container testcontainers.Container) {
	t.Helper()
	state, err := container.State(ctx)
	if err != nil {
		t.Fatalf("read yacy state after stop: %v", err)
	}
	if state.ExitCode == killedOnStopExitCode {
		t.Fatalf(
			"YaCy was killed before it finished writing its index, exit code %d",
			state.ExitCode,
		)
	}
}
