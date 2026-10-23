package memberenvironment_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/memberenvironment"
)

func processFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func environmentFor(
	t *testing.T,
	wiring memberenvironment.Wiring,
	process map[string]string,
) memberenvironment.Environment {
	t.Helper()
	environment, err := memberenvironment.For("corpustext", wiring, nil, processFrom(process))
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	return environment
}

func TestLookupPrefersTheWiredValue(t *testing.T) {
	environment := environmentFor(t, memberenvironment.Wiring{"NATS_URL": "nats://wired"}, nil)

	if got := environment.Lookup("NATS_URL"); got != "nats://wired" {
		t.Errorf("Lookup = %q, want the wired value", got)
	}
}

func TestLookupPrefersTheMemberValueOverTheSharedValue(t *testing.T) {
	environment := environmentFor(t, nil, map[string]string{
		"CORPUSTEXT__LANGUAGES": "de",
		"LANGUAGES":             "en",
	})

	if got := environment.Lookup("LANGUAGES"); got != "de" {
		t.Errorf("Lookup = %q, want the member value", got)
	}
}

func TestLookupFallsBackToTheSharedValue(t *testing.T) {
	environment := environmentFor(t, nil, map[string]string{
		"LANGUAGES":             "en",
		"CORPUSMARKDOWN__LIMIT": "9",
	})

	if got := environment.Lookup("LANGUAGES"); got != "en" {
		t.Errorf("Lookup = %q, want the shared value", got)
	}
	if got := environment.Lookup("LIMIT"); got != "" {
		t.Errorf("Lookup = %q, want no value from another member", got)
	}
}

func TestForRefusesEveryWiredSettingTheOperatorSets(t *testing.T) {
	_, err := memberenvironment.For(
		"corpustext",
		memberenvironment.Wiring{"NATS_URL": "nats://wired", "DATA_DIR": "/data"},
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
	_, err := memberenvironment.For(
		"corpustext",
		nil,
		[]string{"OPS_ADDR"},
		processFrom(map[string]string{"CORPUSTEXT__OPS_ADDR": ":9999"}),
	)
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT__OPS_ADDR") {
		t.Errorf("For = %v, want the unread setting named", err)
	}
}
