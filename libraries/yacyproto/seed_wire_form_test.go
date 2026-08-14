package yacyproto_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func mustPeerName(t *testing.T, name string) yacymodel.PeerName {
	t.Helper()
	parsed, err := yacymodel.ParsePeerName(name)
	if err != nil {
		t.Fatalf("ParsePeerName(%q): %v", name, err)
	}

	return parsed
}

func mustHost(t *testing.T, host string) yacymodel.Host {
	t.Helper()
	parsed, err := yacymodel.ParseHost(host)
	if err != nil {
		t.Fatalf("ParseHost(%q): %v", host, err)
	}

	return parsed
}

func mustTags(t *testing.T, names ...string) yacymodel.PeerTags {
	t.Helper()
	tags := make([]yacymodel.Tag, 0, len(names))
	for _, name := range names {
		tag, err := yacymodel.ParseTag(name)
		if err != nil {
			t.Fatalf("ParseTag(%q): %v", name, err)
		}
		tags = append(tags, tag)
	}
	peerTags, err := yacymodel.NewPeerTags(tags)
	if err != nil {
		t.Fatalf("NewPeerTags: %v", err)
	}

	return peerTags
}

//nolint:revive // a news record is built from all its fields
func mustPeerNews(
	t *testing.T,
	originator yacymodel.Hash,
	category yacymodel.NewsCategory,
	created time.Time,
	received yacymodel.Optional[time.Time],
	distributed int,
	attributes map[string]string,
) yacymodel.PeerNews {
	t.Helper()
	news, err := yacymodel.NewPeerNews(
		originator,
		category,
		created,
		received,
		distributed,
		attributes,
	)
	if err != nil {
		t.Fatalf("NewPeerNews: %v", err)
	}

	return news
}

func mustSeedListURL(t *testing.T, raw string) yacymodel.SeedListURL {
	t.Helper()
	parsed, err := yacymodel.ParseSeedListURL(raw)
	if err != nil {
		t.Fatalf("ParseSeedListURL(%q): %v", raw, err)
	}

	return parsed
}

func fullSeed(t *testing.T) yacymodel.Seed {
	t.Helper()
	offset, err := yacymodel.NewUTCOffset(120)
	if err != nil {
		t.Fatalf("NewUTCOffset: %v", err)
	}
	firstSeen := time.Date(2024, time.March, 2, 10, 20, 30, 0, time.UTC)
	lastSeen := time.Date(2026, time.July, 21, 8, 15, 0, 0, time.UTC)
	disconnected := time.UnixMilli(time.Date(2026, time.July, 20, 0, 0, 0, 0, time.UTC).UnixMilli()).
		UTC()
	created := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	received := time.Date(2026, time.July, 19, 12, 0, 5, 0, time.UTC)

	return yacymodel.Seed{
		Hash:           mustHash(t, "MNOPQRSTUVWX"),
		Name:           mustPeerName(t, "example-peer"),
		PeerType:       yacymodel.PeerSenior,
		PrimaryAddress: yacymodel.Some(mustHost(t, "192.0.2.10")),
		AdditionalAddresses: yacymodel.Some(
			[]yacymodel.Host{mustHost(t, "2001:db8::1"), mustHost(t, "2001:db8::2")},
		),
		Port:            yacymodel.Some(yacymodel.Port(8090)),
		SecurePort:      yacymodel.Some(yacymodel.Port(8443)),
		SeedListAddress: yacymodel.Some(mustSeedListURL(t, "https://example.org/seed.txt")),
		RemotePeerType:  yacymodel.Some(yacymodel.PeerMentor),
		Capabilities: yacymodel.Some(yacymodel.PeerCapabilities{
			DirectConnect:     true,
			AcceptRemoteIndex: true,
			SSLAvailable:      true,
		}),
		Version:           yacymodel.Some(yacymodel.SoftwareVersion{Release: 1.83, Revision: 9000}),
		Tags:              mustTags(t, "news", "search"),
		SolrAvailable:     yacymodel.Some(true),
		FirstSeen:         yacymodel.Some(firstSeen),
		LastSeen:          yacymodel.Some(lastSeen),
		DisconnectedAt:    yacymodel.Some(disconnected),
		UTCOffset:         yacymodel.Some(offset),
		Uptime:            90 * time.Minute,
		IndexingSpeed:     10,
		RetrievalSpeed:    20,
		UplinkSpeed:       30,
		ClientConnectRate: 1.5,
		IndexedWords:      111,
		StoredURLs:        222,
		NoticedURLs:       333,
		RemoteCrawlURLs:   444,
		StoredSeeds:       555,
		WordsSent:         1,
		WordsReceived:     2,
		URLsSent:          3,
		URLsReceived:      4,
		News: yacymodel.Some(mustPeerNews(
			t,
			mustHash(t, "ABCDEFGHIJKL"),
			yacymodel.NewsCrawlStart,
			created,
			yacymodel.Some(received),
			7,
			map[string]string{"startURL": "example.org", "depth": "3"},
		)),
	}
}

func seedFrameWithColumn(t *testing.T, seed yacymodel.Seed, column, value string) string {
	t.Helper()

	row := seedRow(t, seed)
	if !strings.HasSuffix(row, "}") {
		t.Fatalf("seed row is not property form: %q", row)
	}

	return framed('b', yacymodel.Encode([]byte(
		strings.TrimSuffix(row, "}")+","+column+"="+value+"}",
	)))
}

func seedThroughWire(t *testing.T, seed yacymodel.Seed) yacymodel.Seed {
	t.Helper()

	parsed, err := yacyproto.ParseSeed(t.Context(), yacyproto.EncodeSeed(seed))
	if err != nil {
		t.Fatalf("ParseSeed: %v", err)
	}

	return parsed
}

func TestSeedRoundTripsEveryColumn(t *testing.T) {
	t.Parallel()

	want := fullSeed(t)
	if got := seedThroughWire(t, want); !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestParseSeedIgnoresAnUnknownColumn(t *testing.T) {
	t.Parallel()

	seed := sampleSeed(t, "alpha", "example-peer")
	want := seedThroughWire(t, seed)

	got, err := yacyproto.ParseSeed(t.Context(), seedFrameWithColumn(t, seed, "Country", "de"))
	if err != nil {
		t.Fatalf("ParseSeed: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("seed with unknown column = %+v, want %+v", got, want)
	}
}

func TestParseSeedRejectsAnOversizedRow(t *testing.T) {
	t.Parallel()

	seed := sampleSeed(t, "alpha", "example-peer")
	frame := seedFrameWithColumn(t, seed, "Country", strings.Repeat("x", 4096))

	if _, err := yacyproto.ParseSeed(t.Context(), frame); !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("ParseSeed error = %v, want ErrBadSeed", err)
	}
}

func TestParseRemoteSeedRejectsASeedWithNoAddress(t *testing.T) {
	t.Parallel()

	seed := yacymodel.Seed{
		Hash:     mustHash(t, "MNOPQRSTUVWX"),
		Name:     mustPeerName(t, "example-peer"),
		PeerType: yacymodel.PeerSenior,
	}

	_, err := yacyproto.ParseRemoteSeed(t.Context(), yacyproto.EncodeSeed(seed))
	if !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("ParseRemoteSeed error = %v, want ErrBadSeed", err)
	}
}

func TestSeedRoundTripsPeerCapabilities(t *testing.T) {
	t.Parallel()

	cases := []yacymodel.PeerCapabilities{
		{},
		{DirectConnect: true},
		{AcceptRemoteCrawl: true, AcceptRemoteIndex: true},
		{
			DirectConnect:     true,
			AcceptRemoteCrawl: true,
			AcceptRemoteIndex: true,
			RootNode:          true,
			SSLAvailable:      true,
		},
	}
	for _, want := range cases {
		seed := fullSeed(t)
		seed.Capabilities = yacymodel.Some(want)

		got, ok := seedThroughWire(t, seed).Capabilities.Get()
		if !ok || got != want {
			t.Errorf("capabilities %+v round trip to %+v, %v", want, got, ok)
		}
	}
}

func TestSeedRoundTripsTheUTCOffset(t *testing.T) {
	t.Parallel()

	for _, minutes := range []int{0, 120, -120, 330, -60} {
		want, err := yacymodel.NewUTCOffset(minutes)
		if err != nil {
			t.Fatalf("NewUTCOffset(%d): %v", minutes, err)
		}
		seed := fullSeed(t)
		seed.UTCOffset = yacymodel.Some(want)

		got, ok := seedThroughWire(t, seed).UTCOffset.Get()
		if !ok || got != want {
			t.Errorf("offset %d round trips to %+v, %v", minutes, got, ok)
		}
	}
}

func TestParseSeedLeavesAMalformedUTCOffsetAbsent(t *testing.T) {
	t.Parallel()

	seed := sampleSeed(t, "alpha", "example-peer")
	for _, written := range []string{"0100", "*0100", "+ab00"} {
		frame := seedFrameWithColumn(t, seed, "UTC", written)
		got, err := yacyproto.ParseSeed(t.Context(), frame)
		if err != nil {
			t.Fatalf("ParseSeed(UTC=%q): %v", written, err)
		}
		if _, ok := got.UTCOffset.Get(); ok {
			t.Errorf("UTC=%q yields an offset, want none", written)
		}
	}
}

func TestSeedRoundTripsTheSoftwareVersion(t *testing.T) {
	t.Parallel()

	want := yacymodel.SoftwareVersion{Release: 1.83, Revision: 9000}
	seed := fullSeed(t)
	seed.Version = yacymodel.Some(want)

	got, ok := seedThroughWire(t, seed).Version.Get()
	if !ok || got != want {
		t.Fatalf("version round trip = %+v, %v, want %+v", got, ok, want)
	}
}

func TestParseSoftwareVersionRejectsAMalformedVersion(t *testing.T) {
	t.Parallel()

	if _, err := yacyproto.ParseSoftwareVersion("not-a-version"); err == nil {
		t.Fatal("expected error for malformed version")
	}
}

func TestSeedRoundTripsTheSeedTimestamps(t *testing.T) {
	t.Parallel()

	want := time.Date(2026, time.July, 21, 8, 15, 30, 0, time.UTC)
	seed := fullSeed(t)
	seed.LastSeen = yacymodel.Some(want)

	got, ok := seedThroughWire(t, seed).LastSeen.Get()
	if !ok || !got.Equal(want) {
		t.Fatalf("last seen round trip = %v, %v, want %v", got, ok, want)
	}
}

func TestParseSeedLeavesAMalformedTimestampAbsent(t *testing.T) {
	t.Parallel()

	seed := sampleSeed(t, "alpha", "example-peer")
	frame := seedFrameWithColumn(t, seed, "LastSeen", "not-a-date")

	got, err := yacyproto.ParseSeed(t.Context(), frame)
	if err != nil {
		t.Fatalf("ParseSeed: %v", err)
	}
	if _, ok := got.LastSeen.Get(); ok {
		t.Fatal("a malformed timestamp yields a last seen, want none")
	}
}

func TestSeedRoundTripsPeerTags(t *testing.T) {
	t.Parallel()

	seed := fullSeed(t)
	seed.Tags = yacymodel.MatchAllTags()
	if got := seedThroughWire(t, seed).Tags; !got.MatchesAll() {
		t.Errorf("match-all tags round trip to %+v", got)
	}

	want := mustTags(t, "news", "search")
	seed.Tags = want
	if got := seedThroughWire(t, seed).Tags; !reflect.DeepEqual(got, want) {
		t.Errorf("tags round trip = %+v, want %+v", got, want)
	}
}

func TestSeedRoundTripsPeerNews(t *testing.T) {
	t.Parallel()

	want := mustPeerNews(
		t,
		mustHash(t, "ABCDEFGHIJKL"),
		yacymodel.NewsCrawlStart,
		time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC),
		yacymodel.Some(time.Date(2026, time.July, 19, 12, 0, 5, 0, time.UTC)),
		7,
		map[string]string{"startURL": "example.org", "depth": "3"},
	)
	seed := fullSeed(t)
	seed.News = yacymodel.Some(want)

	got, ok := seedThroughWire(t, seed).News.Get()
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("news round trip = %+v, %v, want %+v", got, ok, want)
	}
}

func TestParseSeedRejectsAnOversizedNewsRecord(t *testing.T) {
	t.Parallel()

	seed := fullSeed(t)
	seed.News = yacymodel.Some(mustPeerNews(
		t,
		mustHash(t, "ABCDEFGHIJKL"),
		yacymodel.NewsCrawlStart,
		time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC),
		yacymodel.None[time.Time](),
		0,
		map[string]string{"payload": strings.Repeat("x", 1100)},
	))

	_, err := yacyproto.ParseSeed(t.Context(), yacyproto.EncodeSeed(seed))
	if !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("ParseSeed error = %v, want ErrBadSeed", err)
	}
}

func TestParseSeedRejectsAnUnknownNewsCategory(t *testing.T) {
	t.Parallel()

	record := framed('b', yacymodel.Encode(
		[]byte("{ori=ABCDEFGHIJKL,cat=nonesuch1,cre=20260719120000,dis=0}"),
	))
	frame := seedFrameWithColumn(t, sampleSeed(t, "alpha", "example-peer"), "news", record)

	if _, err := yacyproto.ParseSeed(t.Context(), frame); !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("ParseSeed error = %v, want ErrBadSeed", err)
	}
}
