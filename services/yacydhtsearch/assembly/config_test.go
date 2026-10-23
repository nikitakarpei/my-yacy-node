package yacydhtsearchassembly_test

import (
	"testing"
	"time"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	yacydhtsearch "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
)

func environmentOf(pairs map[string]string) func(string) string {
	return func(key string) string { return pairs[key] }
}

func minimalEnvironment() map[string]string {
	return map[string]string{
		yacydhtsearch.EnvSeedlistURLs:   "http://peer.example/yacy/seedlist.html",
		yacydhtsearch.EnvEgressProxyURL: "http://proxy.example:3128",
	}
}

func TestAServiceConfigShowsUnreadResultsByDefault(t *testing.T) {
	t.Parallel()

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(minimalEnvironment()))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.HideUnreadResults {
		t.Fatal("HideUnreadResults = true, want the unread results shown by default")
	}
}

func TestAServiceConfigFallsBackToTheDocumentedDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(minimalEnvironment()))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.ListenAddr != yacydhtsearch.DefaultListenAddr ||
		cfg.OpsAddr != yacydhtsearch.DefaultOpsAddr {
		t.Fatalf("addresses = %q and %q, want the defaults", cfg.ListenAddr, cfg.OpsAddr)
	}
	if cfg.QueryBudget != yacydhtsearch.DefaultQueryBudget {
		t.Fatalf("query budget = %v, want the default", cfg.QueryBudget)
	}
	if cfg.NetworkRedundancy != yacydhtsearch.DefaultNetworkRedundancy ||
		cfg.PeerCallsInFlight != yacydhtsearch.DefaultPeerCallsInFlight {
		t.Fatalf(
			"network redundancy = %d and peer calls in flight = %d, want the defaults",
			cfg.NetworkRedundancy,
			cfg.PeerCallsInFlight,
		)
	}
	if cfg.URLMetadataCallBudget != yacydhtsearch.DefaultURLMetadataCallBudget ||
		cfg.SearchCallBudget != yacydhtsearch.DefaultSearchCallBudget ||
		cfg.SearchCallHeadersTimeout != yacydhtsearch.DefaultSearchCallHeadersTimeout {
		t.Fatalf(
			"URL metadata call budget = %v, search call budget = %v and headers timeout = %v, "+
				"want the defaults",
			cfg.URLMetadataCallBudget,
			cfg.SearchCallBudget,
			cfg.SearchCallHeadersTimeout,
		)
	}
	if cfg.HedgeDelay != yacydhtsearch.DefaultHedgeDelay {
		t.Fatalf("hedge delay = %v, want the default", cfg.HedgeDelay)
	}
	if cfg.ReplicasCoveringAPartition != yacydhtsearch.DefaultReplicasCoveringAPartition {
		t.Fatalf(
			"replicas covering a partition = %d, want the default",
			cfg.ReplicasCoveringAPartition,
		)
	}
	if cfg.Partitions != 1<<yacydhtsearch.DefaultPartitionExponent {
		t.Fatalf(
			"Partitions = %d, want %d",
			cfg.Partitions,
			1<<yacydhtsearch.DefaultPartitionExponent,
		)
	}
	if len(cfg.SeedlistURLs) != 1 {
		t.Fatalf("SeedlistURLs = %v, want one", cfg.SeedlistURLs)
	}
	if cfg.ServeProfiler {
		t.Fatal("ServeProfiler = true, want the profiler off by default")
	}
	if cfg.PagesReadPerSiteCeiling != yacydhtsearch.DefaultPagesReadPerSiteCeiling {
		t.Fatalf("pages read per site ceiling = %d, want the default", cfg.PagesReadPerSiteCeiling)
	}
	if cfg.PageReadCutoff.PercentOfPages != yacydhtsearch.DefaultPageReadCutoffPercent ||
		cfg.PageReadCutoff.Grace != yacydhtsearch.DefaultPageReadCutoffGrace {
		t.Fatalf("page read cutoff = %+v, want the default", cfg.PageReadCutoff)
	}
	if cfg.URLMetadataLookupCutoff.PercentOfDocuments != yacydhtsearch.DefaultURLMetadataLookupCutoffPercent ||
		cfg.URLMetadataLookupCutoff.Grace != yacydhtsearch.DefaultURLMetadataLookupCutoffGrace {
		t.Fatalf("URL metadata lookup cutoff = %+v, want the default", cfg.URLMetadataLookupCutoff)
	}
	if cfg.QueryWordDocumentAmountLifetime != yacydhtsearch.DefaultQueryWordDocumentAmountLifetime ||
		cfg.QueryWordDocumentAmountsCapacity != yacydhtsearch.DefaultQueryWordDocumentAmountsCapacity {
		t.Fatalf(
			"query word document amount lifetime = %v and capacity = %d, want the defaults",
			cfg.QueryWordDocumentAmountLifetime,
			cfg.QueryWordDocumentAmountsCapacity,
		)
	}
}

func TestTheURLMetadataAskSizingFallsBackToItsDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(minimalEnvironment()))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.URLMetadataAskDocumentsFloor != yacydhtsearch.DefaultURLMetadataAskDocumentsFloor ||
		cfg.URLMetadataAskTargetTime != yacydhtsearch.DefaultURLMetadataAskTargetTime ||
		cfg.URLMetadataAsksPerDocument != yacydhtsearch.DefaultURLMetadataAsksPerDocument {
		t.Fatalf(
			"URL metadata ask documents floor = %d, target time = %v and asks per document = %d, want the defaults",
			cfg.URLMetadataAskDocumentsFloor,
			cfg.URLMetadataAskTargetTime,
			cfg.URLMetadataAsksPerDocument,
		)
	}
}

func TestAnOperatorNamesSeveralSeedlistsInOneSetting(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[yacydhtsearch.EnvSeedlistURLs] = "http://one.example/list, http://two.example/list"

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment))
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
	environment[yacydhtsearch.EnvQueryBudget] = "9s"
	environment[yacydhtsearch.EnvNetworkRedundancy] = "7"
	environment[yacydhtsearch.EnvPeerCallsInFlight] = "9"
	environment[yacydhtsearch.EnvURLMetadataCallBudget] = "4s"
	environment[yacydhtsearch.EnvSearchCallBudget] = "1500ms"
	environment[yacydhtsearch.EnvSearchCallHeadersTimeout] = "0s"
	environment[yacydhtsearch.EnvProbesInFlight] = "12"
	environment[yacydhtsearch.EnvRankedItemsCeiling] = "25"
	environment[yacydhtsearch.EnvDocumentsToMatchCeiling] = "64"
	environment[yacydhtsearch.EnvURLMetadataAskDocumentsCeiling] = "128"
	environment[yacydhtsearch.EnvURLMetadataAskDocumentsFloor] = "16"
	environment[yacydhtsearch.EnvURLMetadataAskTargetTime] = "1500ms"
	environment[yacydhtsearch.EnvURLMetadataAsksPerDocument] = "4"
	environment[yacydhtsearch.EnvReplicasCoveringAPartition] = "2"
	environment[yacydhtsearch.EnvHedgeDelay] = "250ms"
	environment[yacydhtsearch.EnvServeProfiler] = "true"
	environment[yacydhtsearch.EnvHideUnreadResults] = "true"
	environment[yacydhtsearch.EnvPagesReadPerSiteCeiling] = "5"
	environment[yacydhtsearch.EnvPageReadCutoffPercent] = "80"
	environment[yacydhtsearch.EnvPageReadCutoffGrace] = "400ms"
	environment[yacydhtsearch.EnvURLMetadataLookupCutoffPercent] = "75"
	environment[yacydhtsearch.EnvURLMetadataLookupCutoffGrace] = "500ms"
	environment[yacydhtsearch.EnvQueryWordDocumentAmountLifetime] = "90m"
	environment[yacydhtsearch.EnvQueryWordDocumentAmountsCapacity] = "500"

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.QueryBudget != 9*time.Second ||
		cfg.NetworkRedundancy != 7 || cfg.PeerCallsInFlight != 9 ||
		cfg.URLMetadataCallBudget != 4*time.Second ||
		cfg.SearchCallBudget != 1500*time.Millisecond ||
		cfg.SearchCallHeadersTimeout != 0 ||
		cfg.ProbesInFlight != 12 || cfg.RankedItemsCeiling != 25 ||
		cfg.DocumentsToMatchCeiling != 64 ||
		cfg.URLMetadataAskDocumentsCeiling != 128 ||
		cfg.URLMetadataAskDocumentsFloor != 16 ||
		cfg.URLMetadataAskTargetTime != 1500*time.Millisecond ||
		cfg.URLMetadataAsksPerDocument != 4 ||
		cfg.ReplicasCoveringAPartition != 2 || cfg.HedgeDelay != 250*time.Millisecond ||
		!cfg.ServeProfiler || !cfg.HideUnreadResults || cfg.PagesReadPerSiteCeiling != 5 ||
		cfg.PageReadCutoff.PercentOfPages != 80 ||
		cfg.PageReadCutoff.Grace != 400*time.Millisecond ||
		cfg.URLMetadataLookupCutoff.PercentOfDocuments != 75 ||
		cfg.URLMetadataLookupCutoff.Grace != 500*time.Millisecond ||
		cfg.QueryWordDocumentAmountLifetime != 90*time.Minute || cfg.QueryWordDocumentAmountsCapacity != 500 {
		t.Fatalf("config = %+v, want the overrides", cfg)
	}
}

func TestTheServiceRefusesToStartWithoutASeedlist(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	delete(environment, yacydhtsearch.EnvSeedlistURLs)

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted an environment that names no seedlist")
	}
}

func TestTheServiceRefusesToStartWithoutAnEgressProxy(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	delete(environment, yacydhtsearch.EnvEgressProxyURL)

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted an environment that names no egress proxy")
	}
}

func TestTheServiceRefusesAnEgressProxyThatIsNotOne(t *testing.T) {
	t.Parallel()

	for _, proxy := range []string{"ftp://proxy.example", "http://", "://"} {
		environment := minimalEnvironment()
		environment[yacydhtsearch.EnvEgressProxyURL] = proxy
		if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
			t.Fatalf("LoadServiceConfig accepted %q as an egress proxy", proxy)
		}
	}
}

func TestTheServiceRefusesAPartitionExponentTheRingCannotHold(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[yacydhtsearch.EnvPartitionExponent] = "64"

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a partition exponent wider than the ring")
	}
}

func TestTheServiceRefusesMoreReplicasCoveringAPartitionThanTheNetworkRedundancy(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[yacydhtsearch.EnvNetworkRedundancy] = "3"
	environment[yacydhtsearch.EnvReplicasCoveringAPartition] = "4"

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted more replicas covering a partition than the redundancy")
	}
}

func TestTheServiceRefusesABudgetThatIsNotADuration(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[yacydhtsearch.EnvProbeBudget] = "soon"

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a budget that is not a duration")
	}
}

func TestTheServiceRefusesAProfilerSettingThatIsNotABoolean(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[yacydhtsearch.EnvServeProfiler] = "sometimes"

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a profiler setting that is not a boolean")
	}
}

func TestTheServiceRefusesACountThatIsNotOne(t *testing.T) {
	t.Parallel()

	for _, key := range []string{yacydhtsearch.EnvDirectoryCapacity, yacydhtsearch.EnvMaxResponseBytes} {
		environment := minimalEnvironment()
		environment[key] = "-1"
		if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
			t.Fatalf("LoadServiceConfig accepted %s = -1", key)
		}
	}
}

func TestPageReadsLeaveThroughTheEgressProxyUntilTheyAreGivenTheirOwn(t *testing.T) {
	t.Parallel()

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(minimalEnvironment()))
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
	environment[yacydhtsearch.EnvPageReadProxyURL] = "http://archive.example:4750"

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment))
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
		environment[yacydhtsearch.EnvPageReadProxyURL] = proxy
		if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
			t.Fatalf("LoadServiceConfig accepted %q as a page read proxy", proxy)
		}
	}
}

func TestPageReadsTunnelThroughTheProxyUntilToldOtherwise(t *testing.T) {
	t.Parallel()

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(minimalEnvironment()))
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
	environment[yacydhtsearch.EnvPageReadProxyDialMode] = "absolute-url"

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment))
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
	environment[yacydhtsearch.EnvPageReadProxyDialMode] = "carrier-pigeon"

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a dial mode the fetcher cannot speak")
	}
}

func TestInstancesShareStateThroughTheStateServerTheyAreGiven(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[yacydhtsearch.EnvStateRedisURL] = "redis://kvrocks:6666"

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.StateRedis == nil || cfg.StateRedis.Addr != "kvrocks:6666" {
		t.Fatalf("Redis address = %+v, want kvrocks:6666", cfg.StateRedis)
	}
}

func TestEachInstanceKeepsItsOwnStateWithoutAStateServer(t *testing.T) {
	t.Parallel()

	cfg, err := yacydhtsearch.LoadServiceConfig(environmentOf(minimalEnvironment()))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.StateRedis != nil {
		t.Fatalf("Redis address = %+v, want none", cfg.StateRedis)
	}
}

func TestTheServiceRefusesAStateServerAddressThatIsNotRedis(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[yacydhtsearch.EnvStateRedisURL] = "http://kvrocks:6666"

	if _, err := yacydhtsearch.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a Redis address that is not redis://")
	}
}
