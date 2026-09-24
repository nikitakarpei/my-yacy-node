package main_test

import (
	"testing"
	"time"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	main "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/cmd/yacydhtsearch"
)

func environmentOf(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func minimalEnvironment() map[string]string {
	return map[string]string{
		main.EnvSeedlistURLs:   "http://peer.example/yacy/seedlist.html",
		main.EnvEgressProxyURL: "http://proxy.example:3128",
	}
}

func TestAServiceConfigFallsBackToTheDocumentedDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := main.LoadServiceConfig(environmentOf(minimalEnvironment()))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.ListenAddr != main.DefaultListenAddr || cfg.OpsAddr != main.DefaultOpsAddr {
		t.Fatalf("addresses = %q and %q, want the defaults", cfg.ListenAddr, cfg.OpsAddr)
	}
	if cfg.QueryBudget != main.DefaultQueryBudget {
		t.Fatalf("query budget = %v, want the default", cfg.QueryBudget)
	}
	if cfg.NetworkRedundancy != main.DefaultNetworkRedundancy ||
		cfg.PeerCallsInFlight != main.DefaultPeerCallsInFlight {
		t.Fatalf(
			"network redundancy = %d and peer calls in flight = %d, want the defaults",
			cfg.NetworkRedundancy,
			cfg.PeerCallsInFlight,
		)
	}
	if cfg.URLMetadataCallBudget != main.DefaultURLMetadataCallBudget ||
		cfg.SearchCallBudget != main.DefaultSearchCallBudget {
		t.Fatalf(
			"URL metadata call budget = %v and search call budget = %v, want the defaults",
			cfg.URLMetadataCallBudget,
			cfg.SearchCallBudget,
		)
	}
	if cfg.HedgeDelay != main.DefaultHedgeDelay {
		t.Fatalf("hedge delay = %v, want the default", cfg.HedgeDelay)
	}
	if cfg.ReplicasCoveringAPartition != main.DefaultReplicasCoveringAPartition {
		t.Fatalf(
			"replicas covering a partition = %d, want the default",
			cfg.ReplicasCoveringAPartition,
		)
	}
	if cfg.Partitions != 1<<main.DefaultPartitionExponent {
		t.Fatalf("Partitions = %d, want %d", cfg.Partitions, 1<<main.DefaultPartitionExponent)
	}
	if len(cfg.SeedlistURLs) != 1 {
		t.Fatalf("SeedlistURLs = %v, want one", cfg.SeedlistURLs)
	}
	if cfg.ServeProfiler {
		t.Fatal("ServeProfiler = true, want the profiler off by default")
	}
	if cfg.PagesReadPerSite != main.DefaultPagesReadPerSite {
		t.Fatalf("pages read per site = %d, want the default", cfg.PagesReadPerSite)
	}
	if cfg.PageReadCutoff.PercentOfPages != main.DefaultPageReadCutoffPercent ||
		cfg.PageReadCutoff.Grace != main.DefaultPageReadCutoffGrace {
		t.Fatalf("page read cutoff = %+v, want the default", cfg.PageReadCutoff)
	}
	if cfg.QueryWordAmountLifetime != main.DefaultQueryWordAmountLifetime ||
		cfg.QueryWordAmountsCapacity != main.DefaultQueryWordAmountsCapacity {
		t.Fatalf(
			"query word amount lifetime = %v and capacity = %d, want the defaults",
			cfg.QueryWordAmountLifetime,
			cfg.QueryWordAmountsCapacity,
		)
	}
}

func TestAnOperatorNamesSeveralSeedlistsInOneSetting(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvSeedlistURLs] = "http://one.example/list, http://two.example/list"

	cfg, err := main.LoadServiceConfig(environmentOf(environment))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if len(cfg.SeedlistURLs) != 2 || cfg.SeedlistURLs[1] != "http://two.example/list" {
		t.Fatalf("SeedlistURLs = %v, want both seedlists", cfg.SeedlistURLs)
	}
}

func TestAnOperatorOverridesEveryBudgetAndLimit(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvQueryBudget] = "9s"
	environment[main.EnvNetworkRedundancy] = "7"
	environment[main.EnvPeerCallsInFlight] = "9"
	environment[main.EnvURLMetadataCallBudget] = "4s"
	environment[main.EnvSearchCallBudget] = "1500ms"
	environment[main.EnvProbesInFlight] = "12"
	environment[main.EnvRankedItemsCeiling] = "25"
	environment[main.EnvDocumentsToMatchCeiling] = "64"
	environment[main.EnvURLMetadataAskDocumentsCeiling] = "128"
	environment[main.EnvReplicasCoveringAPartition] = "2"
	environment[main.EnvHedgeDelay] = "250ms"
	environment[main.EnvServeProfiler] = "true"
	environment[main.EnvPagesReadPerSite] = "5"
	environment[main.EnvPageReadCutoffPercent] = "80"
	environment[main.EnvPageReadCutoffGrace] = "400ms"
	environment[main.EnvQueryWordAmountLifetime] = "90m"
	environment[main.EnvQueryWordAmountsCapacity] = "500"

	cfg, err := main.LoadServiceConfig(environmentOf(environment))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.QueryBudget != 9*time.Second ||
		cfg.NetworkRedundancy != 7 || cfg.PeerCallsInFlight != 9 ||
		cfg.URLMetadataCallBudget != 4*time.Second ||
		cfg.SearchCallBudget != 1500*time.Millisecond ||
		cfg.ProbesInFlight != 12 || cfg.RankedItemsCeiling != 25 ||
		cfg.DocumentsToMatchCeiling != 64 ||
		cfg.URLMetadataAskDocumentsCeiling != 128 ||
		cfg.ReplicasCoveringAPartition != 2 || cfg.HedgeDelay != 250*time.Millisecond ||
		!cfg.ServeProfiler || cfg.PagesReadPerSite != 5 ||
		cfg.PageReadCutoff.PercentOfPages != 80 ||
		cfg.PageReadCutoff.Grace != 400*time.Millisecond ||
		cfg.QueryWordAmountLifetime != 90*time.Minute || cfg.QueryWordAmountsCapacity != 500 {
		t.Fatalf("config = %+v, want the overrides", cfg)
	}
}

func TestTheServiceRefusesToStartWithoutASeedlist(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	delete(environment, main.EnvSeedlistURLs)

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted an environment that names no seedlist")
	}
}

func TestTheServiceRefusesToStartWithoutAnEgressProxy(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	delete(environment, main.EnvEgressProxyURL)

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted an environment that names no egress proxy")
	}
}

func TestTheServiceRefusesAnEgressProxyThatIsNotOne(t *testing.T) {
	t.Parallel()

	for _, proxy := range []string{"ftp://proxy.example", "http://", "://"} {
		environment := minimalEnvironment()
		environment[main.EnvEgressProxyURL] = proxy
		if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
			t.Fatalf("LoadServiceConfig accepted %q as an egress proxy", proxy)
		}
	}
}

func TestTheServiceRefusesAPartitionExponentTheRingCannotHold(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvPartitionExponent] = "64"

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a partition exponent wider than the ring")
	}
}

func TestTheServiceRefusesMoreReplicasCoveringAPartitionThanTheNetworkRedundancy(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvNetworkRedundancy] = "3"
	environment[main.EnvReplicasCoveringAPartition] = "4"

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted more replicas covering a partition than the redundancy")
	}
}

func TestTheServiceRefusesABudgetThatIsNotADuration(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvProbeBudget] = "soon"

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a budget that is not a duration")
	}
}

func TestTheServiceRefusesAProfilerSettingThatIsNotABoolean(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvServeProfiler] = "sometimes"

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a profiler setting that is not a boolean")
	}
}

func TestTheServiceRefusesACountThatIsNotOne(t *testing.T) {
	t.Parallel()

	for _, key := range []string{main.EnvDirectoryCapacity, main.EnvMaxResponseBytes} {
		environment := minimalEnvironment()
		environment[key] = "-1"
		if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
			t.Fatalf("LoadServiceConfig accepted %s = -1", key)
		}
	}
}

func TestPageReadsLeaveThroughTheEgressProxyUntilTheyAreGivenTheirOwn(t *testing.T) {
	t.Parallel()

	cfg, err := main.LoadServiceConfig(environmentOf(minimalEnvironment()))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.PageReadProxyURL.String() != cfg.EgressProxyURL.String() {
		t.Fatalf(
			"page read proxy = %q, want the egress proxy %q",
			cfg.PageReadProxyURL,
			cfg.EgressProxyURL,
		)
	}
}

func TestPageReadsLeaveThroughTheProxyTheyAreGiven(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvPageReadProxyURL] = "http://archive.example:4750"

	cfg, err := main.LoadServiceConfig(environmentOf(environment))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.PageReadProxyURL.String() != "http://archive.example:4750" {
		t.Fatalf("page read proxy = %q, want the one the environment names", cfg.PageReadProxyURL)
	}
	if cfg.EgressProxyURL.String() != "http://proxy.example:3128" {
		t.Fatalf("egress proxy = %q, want the one the environment names", cfg.EgressProxyURL)
	}
}

func TestTheServiceRefusesAPageReadProxyThatIsNotOne(t *testing.T) {
	t.Parallel()

	for _, proxy := range []string{"ftp://archive.example", "http://", "://"} {
		environment := minimalEnvironment()
		environment[main.EnvPageReadProxyURL] = proxy
		if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
			t.Fatalf("LoadServiceConfig accepted %q as a page read proxy", proxy)
		}
	}
}

func TestPageReadsTunnelThroughTheProxyUntilToldOtherwise(t *testing.T) {
	t.Parallel()

	cfg, err := main.LoadServiceConfig(environmentOf(minimalEnvironment()))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.PageReadProxyDialMode != pagefetchershttp.ProxyDialTunnel {
		t.Fatalf("page read dial mode = %v, want the tunnel default", cfg.PageReadProxyDialMode)
	}
}

func TestPageReadsCanNameTheWholeURLToTheProxy(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvPageReadProxyDialMode] = "absolute-url"

	cfg, err := main.LoadServiceConfig(environmentOf(environment))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.PageReadProxyDialMode != pagefetchershttp.ProxyDialAbsoluteURL {
		t.Fatalf("page read dial mode = %v, want absolute-url", cfg.PageReadProxyDialMode)
	}
}

func TestTheServiceRefusesAPageReadDialModeItCannotSpeak(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvPageReadProxyDialMode] = "carrier-pigeon"

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a dial mode the fetcher cannot speak")
	}
}
