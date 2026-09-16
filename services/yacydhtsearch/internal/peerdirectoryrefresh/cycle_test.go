package peerdirectoryrefresh_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryrefresh"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	responseLimit  = 1 << 20
	directoryLimit = 16
	cooldown       = 5 * time.Second
	refreshEvery   = time.Hour
	probeBudget    = 3 * time.Second
	probesInFlight = 4
	rwiCountAnswer = "version=1.83\nuptime=1200\nresponse=42\n"
	seededHash     = "aaaaaaaaaaaa"
)

type silentDirectoryObserver struct{}

func (silentDirectoryObserver) PeerAdmitted(context.Context, yacymodel.Hash, int)               {}
func (silentDirectoryObserver) PeerAnswered(context.Context, yacymodel.Hash, string, time.Time) {}
func (silentDirectoryObserver) PeerWentSilent(context.Context, yacymodel.Hash)                  {}
func (silentDirectoryObserver) PeerDropped(context.Context, yacymodel.Hash)                     {}
func (silentDirectoryObserver) PeersKnown(context.Context, int, int, int)                       {}

type silentSeedlistObserver struct{}

func (silentSeedlistObserver) SeedlistRead(context.Context, string, int)          {}
func (silentSeedlistObserver) SeedlistUnreachable(context.Context, string, error) {}
func (silentSeedlistObserver) SeedlistUnreadable(context.Context, string, error)  {}

type earnedPresences map[peeranswerhistory.PeerAtAddress]time.Duration

func (presences earnedPresences) EarnedPresenceOf(
	_ context.Context,
	peerAtAddress peeranswerhistory.PeerAtAddress,
) time.Duration {
	return presences[peerAtAddress]
}

type stalestFirst struct{}

func (stalestFirst) StalestPeers(
	_ context.Context,
	known []peerdirectory.KnownPeer,
	_ int,
) []yacymodel.Hash {
	return []yacymodel.Hash{known[0].Hash}
}

func peerAnsweringProbes(t *testing.T, status int) (host, port string) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(status)
			_, _ = writer.Write([]byte(rwiCountAnswer))
		},
	))
	t.Cleanup(server.Close)

	address := strings.TrimPrefix(server.URL, "http://")
	split := strings.LastIndex(address, ":")

	return address[:split], address[split+1:]
}

func seedlistNaming(t *testing.T, host, port string, alsoAt ...string) string {
	t.Helper()

	line := "Hash=" + seededHash + ",Name=holder,PeerType=senior,IP=" + host + ",Port=" + port
	if len(alsoAt) > 0 {
		line += ",IP6=" + strings.Join(alsoAt, "|")
	}
	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(line))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func refreshOver(
	t *testing.T,
	seedlistURL string,
	directory *peerdirectory.Directory,
	presence earnedPresences,
) peerdirectoryrefresh.Cycle {
	t.Helper()

	return peerdirectoryrefresh.New(
		yacyseedlist.New(
			http.DefaultClient,
			[]string{seedlistURL},
			responseLimit,
			silentSeedlistObserver{},
		),
		directory,
		peerlivenesswire.New(
			http.DefaultClient, "freeworld", peerlivenesswire.PeerLivenessObservers{},
		),
		presence,
		peerdirectoryrefresh.ProbeLimits{
			ProbeBudget:    probeBudget,
			ProbesInFlight: probesInFlight,
		},
	)
}

func directoryOf(t *testing.T) *peerdirectory.Directory {
	t.Helper()

	return peerdirectory.New(
		directoryLimit,
		cooldown,
		time.Now,
		stalestFirst{},
		silentDirectoryObserver{},
	)
}

func TestOneRefreshMakesASeededPeerAskable(t *testing.T) {
	t.Parallel()

	host, port := peerAnsweringProbes(t, http.StatusOK)
	directory := directoryOf(t)

	refreshOver(
		t,
		seedlistNaming(t, host, port),
		directory,
		earnedPresences{},
	).RefreshOnce(t.Context())

	askable := directory.AskablePeers(t.Context())
	if len(askable) != 1 || askable[0].Address != "http://"+host+":"+port {
		t.Fatalf("AskablePeers = %+v, want the address that answered", askable)
	}
}

func TestAPeerThatAnswersNoProbeStaysUnaskable(t *testing.T) {
	t.Parallel()

	host, port := peerAnsweringProbes(t, http.StatusForbidden)
	directory := directoryOf(t)

	refreshOver(
		t,
		seedlistNaming(t, host, port),
		directory,
		earnedPresences{},
	).RefreshOnce(t.Context())

	if known := directory.KnownPeers(t.Context()); len(known) != 1 {
		t.Fatalf("KnownPeers = %+v, want the seeded peer", known)
	}
	if askable := directory.AskablePeers(t.Context()); len(askable) != 0 {
		t.Fatalf("AskablePeers = %+v, want none", askable)
	}
}

func TestTheCycleRefreshesUntilTheServiceStops(t *testing.T) {
	t.Parallel()

	host, port := peerAnsweringProbes(t, http.StatusOK)
	directory := directoryOf(t)
	cycle := refreshOver(
		t, seedlistNaming(t, host, port), directory, earnedPresences{},
	)

	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		cycle.Run(ctx, refreshEvery)
	}()

	for len(directory.AskablePeers(t.Context())) == 0 {
		time.Sleep(time.Millisecond)
	}
	stop()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the service stopped")
	}
}

func TestTheAddressAPeerHasEarnedPresenceAtIsProbedFirst(t *testing.T) {
	t.Parallel()

	host, port := peerAnsweringProbes(t, http.StatusOK)
	presentAddress := "http://localhost:" + port
	directory := directoryOf(t)

	refreshOver(
		t,
		seedlistNaming(t, host, port, "localhost"),
		directory,
		earnedPresences{
			peeranswerhistory.PeerAtAddress{
				Hash:    peerHash(t),
				Address: presentAddress,
			}: time.Hour,
		},
	).RefreshOnce(t.Context())

	askable := directory.AskablePeers(t.Context())
	if len(askable) != 1 || askable[0].Address != presentAddress {
		t.Fatalf("AskablePeers = %+v, want the address the peer earned presence at", askable)
	}
}

func peerHash(t *testing.T) yacymodel.Hash {
	t.Helper()

	parsed, err := yacymodel.ParseHash(seededHash)
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return parsed
}
