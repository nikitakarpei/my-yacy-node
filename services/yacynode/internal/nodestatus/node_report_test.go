package nodestatus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeidentity"
)

type stubCounter struct {
	rwi  int
	refs int
	urls int
	err  error
}

func (c stubCounter) RWICount(context.Context) (int, error)           { return c.rwi, c.err }
func (c stubCounter) ReferencedURLCount(context.Context) (int, error) { return c.refs, c.err }
func (c stubCounter) Count(context.Context) (int, error)              { return c.urls, c.err }

func mustPeerName(name string) yacymodel.PeerName {
	peerName, err := yacymodel.ParsePeerName(name)
	if err != nil {
		panic(err)
	}

	return peerName
}

func testIdentity() nodeidentity.Identity {
	return nodeidentity.Identity{
		Hash:        yacymodel.WordHash("self"),
		NetworkName: "freeworld",
		Name:        mustPeerName("node"),
		Host:        "192.0.2.1",
		Port:        8090,
		Flags:       yacymodel.PeerCapabilities{},
		Version:     "1.2",
	}
}

func clockAt(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func reportAt(start time.Time, elapsed time.Duration, rwi, urls stubCounter) nodeReport {
	id := testIdentity()
	id.Start = start

	return newReport(id, clockAt(start.Add(elapsed)), rwi, urls)
}

func TestSelfSeedRefreshesDynamicFields(t *testing.T) {
	start := time.Date(2026, time.June, 22, 10, 0, 0, 0, time.UTC)
	counts := stubCounter{rwi: 7, urls: 3}
	report := reportAt(start, 90*time.Minute, counts, counts)

	seed := report.SelfSeed(context.Background())

	if seed.Uptime != 90*time.Minute {
		t.Fatalf("Uptime = %v, want 90m", seed.Uptime)
	}
	if seed.IndexedWords != 7 {
		t.Fatalf("IndexedWords = %d, want 7", seed.IndexedWords)
	}
	if seed.StoredURLs != 3 {
		t.Fatalf("StoredURLs = %d, want 3", seed.StoredURLs)
	}
	if _, ok := seed.LastSeen.Get(); !ok {
		t.Fatal("LastSeen unset")
	}
	if _, ok := seed.UTCOffset.Get(); !ok {
		t.Fatal("UTCOffset unset")
	}
}

func TestSelfSeedKeepsIdentityFields(t *testing.T) {
	start := time.Date(2026, time.June, 22, 10, 0, 0, 0, time.UTC)
	counts := stubCounter{}
	report := reportAt(start, 0, counts, counts)

	seed := report.SelfSeed(context.Background())

	if seed.Hash != yacymodel.WordHash("self") {
		t.Fatalf("Hash = %q, want self hash", seed.Hash)
	}
	if seed.Name != mustPeerName("node") {
		t.Fatalf("Name = %q, want node", seed.Name)
	}
	if port, _ := seed.Port.Get(); port != yacymodel.Port(8090) {
		t.Fatalf("Port = %d, want 8090", port)
	}
	if seed.PeerType != yacymodel.PeerSenior {
		t.Fatalf("PeerType = %q, want senior", seed.PeerType)
	}
	host, ok := seed.PrimaryAddress.Get()
	if !ok || host.String() != "192.0.2.1" {
		t.Fatalf("IP = %q (set %v), want 192.0.2.1", host, ok)
	}
}

func TestSelfSeedCountErrorsReportZero(t *testing.T) {
	start := time.Date(2026, time.June, 22, 10, 0, 0, 0, time.UTC)
	counts := stubCounter{rwi: 5, urls: 9, err: errors.New("boom")}
	report := reportAt(start, time.Hour, counts, counts)

	seed := report.SelfSeed(context.Background())

	if seed.IndexedWords != 0 {
		t.Fatalf("IndexedWords = %d, want 0 on error", seed.IndexedWords)
	}
	if seed.StoredURLs != 0 {
		t.Fatalf("StoredURLs = %d, want 0 on error", seed.StoredURLs)
	}
}

func TestHeaderReportsVersionAndUptime(t *testing.T) {
	start := time.Date(2026, time.June, 22, 10, 0, 0, 0, time.UTC)
	report := reportAt(start, 45*time.Minute, stubCounter{}, stubCounter{})

	ctx := context.Background()

	if got := report.Version(ctx); got != "1.2" {
		t.Fatalf("Version = %q, want 1.2", got)
	}
	if got := report.Uptime(ctx); got != 45 {
		t.Fatalf("Uptime = %d, want 45", got)
	}
}
