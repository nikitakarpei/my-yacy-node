//go:build e2e

// Package nodepeer starts and configures the node-under-test, alone or as a
// fleet.
package nodepeer

import (
	"context"
	"strconv"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerlog"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/containerurl"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/httpprobe"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/requiredimage"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/test/e2e/peerclient"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	// MinConnectedPeersForDHT is the connected-peer count below which YaCy
	// calls the network too small and distributes no RWI at all.
	MinConnectedPeersForDHT = 33

	// FleetSize keeps the fleet above MinConnectedPeersForDHT while YaCy holds
	// a peer out of its connected set between two of that peer's hellos.
	FleetSize = MinConnectedPeersForDHT + 3

	envNodeImage = "YACY_NODE_IMAGE"
)

type Config struct {
	NetworkName  string
	Alias        string
	Hash         yacymodel.Hash
	SeedlistURL  string
	Distribution DistributionConfig
}

// DistributionConfig turns on the node's outbound RWI distribution cycle.
// The zero value leaves distribution disabled, matching the node's own default.
type DistributionConfig struct {
	Enabled               bool
	Redundancy            int
	PartitionExponent     int
	PostingsPerBatch      int
	CycleInterval         time.Duration
	DrainBudget           time.Duration
	LongestOfferInterval  time.Duration
	ShortestOfferInterval time.Duration
	MinReachablePeers     int
}

func Start(
	t *testing.T,
	ctx context.Context,
	probe *httpprobe.Probe,
	cfg Config,
) (testcontainers.Container, string) {
	t.Helper()
	env := map[string]string{
		"YACY_INITIAL_PEER_HASH":         cfg.Hash.String(),
		"YACY_PEER_NAME":                 cfg.Alias,
		"YACY_NETWORK_NAME":              yacyproto.DefaultNetwork,
		"YACY_PEER_ADDR":                 ":" + peerclient.Port,
		"YACY_ADVERTISE_HOST":            cfg.Alias,
		"YACY_ADVERTISE_PORT":            peerclient.Port,
		"YACY_DATA_DIR":                  "/tmp/data",
		"YACY_ANNOUNCE_INTERVAL":         "10s",
		"YACY_REACHABLE_ROSTER_CAPACITY": strconv.Itoa(FleetSize + 8),
		"EGRESS_PROXY_URL":               egressproxy.NetworkURL(),
		"LOG_LEVEL":                      "debug",
	}
	if cfg.SeedlistURL != "" {
		env["YACY_SEEDLIST_URLS"] = cfg.SeedlistURL
	}
	if cfg.Distribution.Enabled {
		env["YACY_DISTRIBUTION_ENABLED"] = "true"
		env["YACY_DISTRIBUTION_REDUNDANCY"] = strconv.Itoa(cfg.Distribution.Redundancy)
		env["YACY_DISTRIBUTION_PARTITION_EXPONENT"] = strconv.Itoa(
			cfg.Distribution.PartitionExponent,
		)
		env["YACY_DISTRIBUTION_POSTINGS_PER_BATCH"] = strconv.Itoa(
			cfg.Distribution.PostingsPerBatch,
		)
		env["YACY_DISTRIBUTION_CYCLE_INTERVAL"] = cfg.Distribution.CycleInterval.String()
		env["YACY_DISTRIBUTION_DRAIN_BUDGET"] = cfg.Distribution.DrainBudget.String()
		env["YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL"] = cfg.Distribution.LongestOfferInterval.String()
		env["YACY_DISTRIBUTION_SHORTEST_OFFER_INTERVAL"] = cfg.Distribution.ShortestOfferInterval.String()
		if cfg.Distribution.MinReachablePeers > 0 {
			env["YACY_DISTRIBUTION_MIN_REACHABLE_PEERS"] = strconv.Itoa(
				cfg.Distribution.MinReachablePeers,
			)
		}
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started: true,
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          image(t),
			Name:           cfg.Alias,
			ExposedPorts:   []string{peerclient.ExposedPort},
			Env:            env,
			Networks:       []string{cfg.NetworkName},
			NetworkAliases: map[string][]string{cfg.NetworkName: {cfg.Alias}},
			Tmpfs:          map[string]string{"/tmp": "rw,mode=1777"},
			HostConfigModifier: func(hostConfig *dockercontainer.HostConfig) {
				hostConfig.ReadonlyRootfs = true
				hostConfig.CapDrop = []string{"ALL"}
				hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "no-new-privileges")
			},
		},
	})
	if err != nil {
		t.Fatalf("start node container from Dockerfile: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	containerlog.DumpOnFailure(t, "node", container)
	nodeURL := containerurl.HostURL(t, ctx, container, peerclient.ExposedPort)
	if !pollwait.For(20*time.Second, func() bool {
		return probe.OK(ctx, nodeURL+"/yacy/query.html?object=rwicount")
	}) {
		t.Fatalf("node %s never became reachable from the host", cfg.Alias)
	}
	return container, nodeURL
}

func image(t *testing.T) string {
	t.Helper()
	return requiredimage.FromEnv(t, envNodeImage, "node", "e2e")
}
