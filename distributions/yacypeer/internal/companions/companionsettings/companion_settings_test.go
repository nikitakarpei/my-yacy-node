package companionsettings_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
)

type toldChoices struct {
	choices []companionsettings.Choice
}

func (t *toldChoices) ChoiceMade(_ context.Context, choice companionsettings.Choice) {
	t.choices = append(t.choices, choice)
}

func opened(
	t *testing.T,
	directory string,
	observers ...companionsettings.Observer,
) *companionsettings.Settings {
	t.Helper()
	settings, err := companionsettings.Open(context.Background(), directory, observers...)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = settings.Close() })
	return settings
}

func TestCompanionsFetchUntilAChoiceIsMade(t *testing.T) {
	told := &toldChoices{}
	want := companionsettings.Choice{CompanionsFetch: true}

	settings := opened(t, t.TempDir(), told)

	if settings.Choice() != want {
		t.Errorf("Choice = %+v, want %+v", settings.Choice(), want)
	}
	if len(told.choices) != 1 || told.choices[0] != want {
		t.Errorf("told %+v, want the default choice once", told.choices)
	}
}

func TestTheSettingsKeepTheChoiceAcrossARestart(t *testing.T) {
	directory := t.TempDir()
	choice := companionsettings.Choice{CompanionsFetch: true, PeerFillsIn: true}
	first, err := companionsettings.Open(context.Background(), directory)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	first.Choose(context.Background(), choice)
	if err := first.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	told := &toldChoices{}

	settings := opened(t, directory, told)

	if settings.Choice() != choice {
		t.Errorf("Choice = %+v, want %+v", settings.Choice(), choice)
	}
	if len(told.choices) != 1 || told.choices[0] != choice {
		t.Errorf("told %+v, want the kept choice", told.choices)
	}
}

func TestTheSettingsTellEachChoice(t *testing.T) {
	told := &toldChoices{}
	settings := opened(t, t.TempDir(), told)
	choice := companionsettings.Choice{PeerFillsIn: true}

	settings.Choose(context.Background(), choice)

	if len(told.choices) != 2 || told.choices[1] != choice {
		t.Errorf("told %+v, want %+v last", told.choices, choice)
	}
}
