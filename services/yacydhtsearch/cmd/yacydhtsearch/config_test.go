package main_test

import (
	"testing"
	"time"

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
	if cfg.QueryBudget != main.DefaultQueryBudget ||
		cfg.PeerChoiceCooldown != main.DefaultPeerChoiceCooldown {
		t.Fatalf(
			"budgets = %v and %v, want the defaults",
			cfg.QueryBudget,
			cfg.PeerChoiceCooldown,
		)
	}
	if cfg.Partitions != 1<<main.DefaultPartitionExponent {
		t.Fatalf("Partitions = %d, want %d", cfg.Partitions, 1<<main.DefaultPartitionExponent)
	}
	if len(cfg.SeedlistURLs) != 1 {
		t.Fatalf("SeedlistURLs = %v, want one", cfg.SeedlistURLs)
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
	environment[main.EnvPeerChoiceCooldown] = "7s"
	environment[main.EnvPeerCallsPerQuery] = "7"
	environment[main.EnvProbesInFlight] = "12"
	environment[main.EnvRankedItemsCeiling] = "25"

	cfg, err := main.LoadServiceConfig(environmentOf(environment))
	if err != nil {
		t.Fatalf("load service config: %v", err)
	}
	if cfg.QueryBudget != 9*time.Second || cfg.PeerChoiceCooldown != 7*time.Second ||
		cfg.PeerCallsPerQuery != 7 || cfg.ProbesInFlight != 12 || cfg.RankedItemsCeiling != 25 {
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

func TestTheServiceRefusesABudgetThatIsNotADuration(t *testing.T) {
	t.Parallel()

	environment := minimalEnvironment()
	environment[main.EnvProbeBudget] = "soon"

	if _, err := main.LoadServiceConfig(environmentOf(environment)); err == nil {
		t.Fatal("LoadServiceConfig accepted a budget that is not a duration")
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
