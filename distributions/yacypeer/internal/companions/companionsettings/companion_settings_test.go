package companionsettings_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionsettings"
)

type toldFetchings struct {
	fetchings []companionsettings.Fetching
}

func (t *toldFetchings) FetchingChanged(_ context.Context, fetching companionsettings.Fetching) {
	t.fetchings = append(t.fetchings, fetching)
}

func opened(
	t *testing.T,
	directory string,
	observers ...companionsettings.Observer,
) *companionsettings.Keeper {
	t.Helper()
	keeper, err := companionsettings.Open(context.Background(), directory, observers...)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(keeper.Close)
	return keeper
}

func TestCompanionsFetchUntilTheOperatorChangesTheFetching(t *testing.T) {
	told := &toldFetchings{}
	want := companionsettings.Fetching{CompanionsFetch: true}

	keeper := opened(t, t.TempDir(), told)

	if keeper.Fetching() != want {
		t.Errorf("Fetching = %+v, want %+v", keeper.Fetching(), want)
	}
	if len(told.fetchings) != 1 || told.fetchings[0] != want {
		t.Errorf("told %+v, want the default fetching once", told.fetchings)
	}
}

func TestTheKeeperKeepsTheFetchingAcrossARestart(t *testing.T) {
	directory := t.TempDir()
	fetching := companionsettings.Fetching{CompanionsFetch: true, PeerFillsIn: true}
	first, err := companionsettings.Open(context.Background(), directory)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	first.ChangeFetching(context.Background(), fetching)
	first.Close()
	told := &toldFetchings{}

	keeper := opened(t, directory, told)

	if keeper.Fetching() != fetching {
		t.Errorf("Fetching = %+v, want %+v", keeper.Fetching(), fetching)
	}
	if len(told.fetchings) != 1 || told.fetchings[0] != fetching {
		t.Errorf("told %+v, want the kept fetching", told.fetchings)
	}
}

func TestTheKeeperTellsEachChange(t *testing.T) {
	told := &toldFetchings{}
	keeper := opened(t, t.TempDir(), told)
	fetching := companionsettings.Fetching{PeerFillsIn: true}

	keeper.ChangeFetching(context.Background(), fetching)

	if len(told.fetchings) != 2 || told.fetchings[1] != fetching {
		t.Errorf("told %+v, want %+v last", told.fetchings, fetching)
	}
}
