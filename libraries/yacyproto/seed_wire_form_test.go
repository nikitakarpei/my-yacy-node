package yacyproto

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
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

func TestSeedWireRoundTrip(t *testing.T) {
	want := fullSeed(t)

	got, err := ParseSeed(context.Background(), EncodeSeed(want))
	if err != nil {
		t.Fatalf("ParseSeed: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestSeedWireIgnoresUnknownKey(t *testing.T) {
	base := seedWireForm{
		properties: map[string]string{
			seedColHash:     "MNOPQRSTUVWX",
			seedColName:     "example-peer",
			seedColPeerType: yacymodel.PeerSenior.String(),
		},
		columns: []string{seedColHash, seedColName, seedColPeerType},
	}
	withUnknown := seedWireForm{
		properties: map[string]string{
			seedColHash:     "MNOPQRSTUVWX",
			seedColName:     "example-peer",
			seedColPeerType: yacymodel.PeerSenior.String(),
			"Country":       "de",
		},
		columns: []string{seedColHash, seedColName, seedColPeerType, "Country"},
	}

	want, err := ParseSeed(context.Background(), base.framed())
	if err != nil {
		t.Fatalf("ParseSeed(base): %v", err)
	}
	got, err := ParseSeed(context.Background(), withUnknown.framed())
	if err != nil {
		t.Fatalf("ParseSeed(withUnknown): %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseSeed with unknown key = %+v, want %+v", got, want)
	}
}

func TestSeedWireRejectsOversizedForm(t *testing.T) {
	framed := seedWireForm{
		properties: map[string]string{
			seedColHash:     "MNOPQRSTUVWX",
			seedColName:     "example-peer",
			seedColPeerType: yacymodel.PeerSenior.String(),
			seedColTags:     strings.Repeat("x", seedMaxPlainBytes),
		},
		columns: []string{seedColHash, seedColName, seedColPeerType, seedColTags},
	}.framed()

	_, err := ParseSeed(context.Background(), framed)
	if !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("ParseSeed error = %v, want ErrBadSeed", err)
	}
}

func TestRemoteSeedRejectsMissingAddress(t *testing.T) {
	seed := yacymodel.Seed{
		Hash:     mustHash(t, "MNOPQRSTUVWX"),
		Name:     mustPeerName(t, "example-peer"),
		PeerType: yacymodel.PeerSenior,
	}

	_, err := ParseRemoteSeed(context.Background(), EncodeSeed(seed))
	if !errors.Is(err, yacymodel.ErrBadSeed) {
		t.Fatalf("ParseRemoteSeed error = %v, want ErrBadSeed", err)
	}
}

func TestPeerCapabilitiesWireRoundTrip(t *testing.T) {
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
	codec := peerCapabilitiesWireCodec{}
	for _, want := range cases {
		if got := codec.decode(codec.encode(want)); got != want {
			t.Errorf("round-trip %+v = %+v", want, got)
		}
	}
}

func TestUTCOffsetWireRoundTrip(t *testing.T) {
	codec := utcOffsetWireCodec{}
	for _, minutes := range []int{0, 120, -120, 330, -60} {
		want, err := yacymodel.NewUTCOffset(minutes)
		if err != nil {
			t.Fatalf("NewUTCOffset(%d): %v", minutes, err)
		}
		got, err := codec.decode(codec.encode(want))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got != want {
			t.Errorf("round-trip %d = %+v", minutes, got)
		}
	}
}

func TestUTCOffsetWireRejectsMalformed(t *testing.T) {
	codec := utcOffsetWireCodec{}
	for _, text := range []string{"", "0100", "*0100", "+ab00"} {
		if _, err := codec.decode(text); !errors.Is(err, yacymodel.ErrBadUTCOffset) {
			t.Errorf("decode(%q) error = %v, want ErrBadUTCOffset", text, err)
		}
	}
}

func TestSoftwareVersionWireRoundTrip(t *testing.T) {
	codec := softwareVersionWireCodec{}
	want := yacymodel.SoftwareVersion{Release: 1.83, Revision: 9000}
	got, err := codec.decode(codec.encode(want))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got != want {
		t.Errorf("round-trip = %+v, want %+v", got, want)
	}
	if _, err := codec.decode("not-a-version"); !errors.Is(err, errBadSoftwareVersion) {
		t.Errorf("decode(bad) error = %v, want errBadSoftwareVersion", err)
	}
}

func TestSeedTimestampWireRoundTrip(t *testing.T) {
	codec := seedTimestampWireCodec{}
	want := time.Date(2026, time.July, 21, 8, 15, 30, 0, time.UTC)
	got, ok := codec.decode(codec.encode(want))
	if !ok {
		t.Fatalf("decode failed")
	}
	if !got.Equal(want) {
		t.Errorf("round-trip = %v, want %v", got, want)
	}
	if _, ok := codec.decode("not-a-date"); ok {
		t.Errorf("decode(bad) ok = true, want false")
	}
}

func TestPeerTagsWireRoundTrip(t *testing.T) {
	codec := peerTagsWireCodec{}

	if got := codec.encode(yacymodel.MatchAllTags()); got != peerTagsWildcard {
		t.Errorf("encode(match-all) = %q, want %q", got, peerTagsWildcard)
	}
	if tags, err := codec.decode(peerTagsWildcard); err != nil || !tags.MatchesAll() {
		t.Errorf("decode(wildcard) = %+v, %v", tags, err)
	}

	want := mustTags(t, "news", "search")
	got, err := codec.decode(codec.encode(want))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip = %+v, want %+v", got, want)
	}
}

func TestPeerNewsWireRoundTrip(t *testing.T) {
	codec := peerNewsWireCodec{}
	want := mustPeerNews(
		t,
		mustHash(t, "ABCDEFGHIJKL"),
		yacymodel.NewsCrawlStart,
		time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC),
		yacymodel.Some(time.Date(2026, time.July, 19, 12, 0, 5, 0, time.UTC)),
		7,
		map[string]string{"startURL": "example.org", "depth": "3"},
	)
	got, err := codec.decode(context.Background(), codec.encode(want))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestPeerNewsWireRejectsOversizeRecord(t *testing.T) {
	news := mustPeerNews(
		t,
		mustHash(t, "ABCDEFGHIJKL"),
		yacymodel.NewsCrawlStart,
		time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC),
		yacymodel.None[time.Time](),
		0,
		map[string]string{"payload": strings.Repeat("x", 1100)},
	)
	codec := peerNewsWireCodec{}
	if _, err := codec.decode(context.Background(), codec.encode(news)); !errors.Is(
		err,
		yacymodel.ErrBadPeerNews,
	) {
		t.Fatalf("decode error = %v, want ErrBadPeerNews", err)
	}
}

func TestPeerNewsWireRejectsUnknownCategory(t *testing.T) {
	framed := encodeBase64WireForm("{ori=ABCDEFGHIJKL,cat=nonesuch1,cre=20260719120000,dis=0}")
	if _, err := (peerNewsWireCodec{}).decode(
		context.Background(),
		framed,
	); !errors.Is(
		err,
		yacymodel.ErrBadPeerNews,
	) {
		t.Fatalf("decode error = %v, want ErrBadPeerNews", err)
	}
}
