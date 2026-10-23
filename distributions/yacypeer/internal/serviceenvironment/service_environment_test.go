package serviceenvironment_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

func processFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func environmentFor(
	t *testing.T,
	peer serviceenvironment.PeerSettings,
	process map[string]string,
) serviceenvironment.Environment {
	t.Helper()
	environment, err := serviceenvironment.For("corpustext", peer, nil, processFrom(process))
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	return environment
}

func TestLookupPrefersTheWiredValue(t *testing.T) {
	environment := environmentFor(
		t,
		serviceenvironment.PeerSettings{Wired: map[string]string{"NATS_URL": "nats://wired"}},
		nil,
	)

	if got := environment.Lookup("NATS_URL"); got != "nats://wired" {
		t.Errorf("Lookup = %q, want the wired value", got)
	}
}

func TestLookupPrefersTheServiceValueOverTheSharedValue(t *testing.T) {
	environment := environmentFor(t, serviceenvironment.PeerSettings{}, map[string]string{
		"CORPUSTEXT__LANGUAGES": "de",
		"LANGUAGES":             "en",
	})

	if got := environment.Lookup("LANGUAGES"); got != "de" {
		t.Errorf("Lookup = %q, want the service value", got)
	}
}

func TestLookupFallsBackToTheSharedValue(t *testing.T) {
	environment := environmentFor(t, serviceenvironment.PeerSettings{}, map[string]string{
		"LANGUAGES":             "en",
		"CORPUSMARKDOWN__LIMIT": "9",
	})

	if got := environment.Lookup("LANGUAGES"); got != "en" {
		t.Errorf("Lookup = %q, want the shared value", got)
	}
	if got := environment.Lookup("LIMIT"); got != "" {
		t.Errorf("Lookup = %q, want no value from another service", got)
	}
}

func TestForRefusesEveryWiredSettingTheOperatorSets(t *testing.T) {
	_, err := serviceenvironment.For(
		"corpustext",
		serviceenvironment.PeerSettings{
			Wired: map[string]string{"NATS_URL": "nats://wired", "DATA_DIR": "/data"},
		},
		nil,
		processFrom(map[string]string{
			"NATS_URL":             "nats://elsewhere",
			"CORPUSTEXT__DATA_DIR": "/elsewhere",
		}),
	)
	if err == nil {
		t.Fatal("For accepted overridden wiring, want an error")
	}
	for _, name := range []string{"NATS_URL", "CORPUSTEXT__DATA_DIR"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestForRefusesEveryUnreadSettingTheOperatorSets(t *testing.T) {
	_, err := serviceenvironment.For(
		"corpustext",
		serviceenvironment.PeerSettings{},
		[]string{"OPS_ADDR"},
		processFrom(map[string]string{"CORPUSTEXT__OPS_ADDR": ":9999"}),
	)
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT__OPS_ADDR") {
		t.Errorf("For = %v, want the unread setting named", err)
	}
}

func TestLookupPrefersTheOperatorValueOverTheDefault(t *testing.T) {
	environment := environmentFor(
		t,
		serviceenvironment.PeerSettings{
			Defaults: map[string]string{"LISTEN_ADDR": ":8093", "OPS_ADDR": ":9093"},
		},
		map[string]string{"LISTEN_ADDR": ":7000", "CORPUSTEXT__OPS_ADDR": ":7001"},
	)

	if got := environment.Lookup("LISTEN_ADDR"); got != ":7000" {
		t.Errorf("Lookup = %q, want the shared value", got)
	}
	if got := environment.Lookup("OPS_ADDR"); got != ":7001" {
		t.Errorf("Lookup = %q, want the service value", got)
	}
}

func TestLookupFallsBackToTheDefault(t *testing.T) {
	environment := environmentFor(
		t,
		serviceenvironment.PeerSettings{Defaults: map[string]string{"LISTEN_ADDR": ":8093"}},
		nil,
	)

	if got := environment.Lookup("LISTEN_ADDR"); got != ":8093" {
		t.Errorf("Lookup = %q, want the default", got)
	}
}
