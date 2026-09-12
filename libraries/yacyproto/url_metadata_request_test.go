package yacyproto_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const realYaCyURLMetadataBody = `<?xml version="1.0"?>
<rss>
<yacy version="1.924">
<iam>Dsji4JNiyZFo</iam>
<uptime>8714</uptime>
<mytime>20260910114634</mytime>
<response>ok</response>
</yacy>
<channel>
<title></title>
<description></description>
<pubDate></pubDate>
<item>
<title>NinjaZumbi&apos;s Dojo - A Blogh about anything</title>
<link>https://www.ninjazumbi.com/</link>
<referrer>https://lemmy.world/post/60585</referrer>
<description>NinjaZumbi&apos;s Dojo - A Blogh about anything</description>
<author>Anarch157a</author>
<pubDate>20251116171731</pubDate>
<guid isPermaLink="false">Q_ylfl--9bK5</guid>
</item>
<item>
<title>Userbenchmark&apos;s world</title>
<link>https://userbenchmarksweb.neocities.org/</link>
<referrer>https://hotlinewebring.club/</referrer>
<description>Userbenchmark&apos;s world</description>
<author></author>
<pubDate>20220505025041</pubDate>
<guid isPermaLink="false">eVZzCn--SAOx</guid>
</item>
</channel>
</rss>`

func TestURLMetadataRequestFormCarriesTheNamesUnseparated(t *testing.T) {
	request := yacyproto.URLMetadataRequest{
		NetworkName: "freeworld",
		URLs: []yacymodel.URLHash{
			mustParseURLHash(t, "Q_ylfl--9bK5"),
			mustParseURLHash(t, "eVZzCn--SAOx"),
		},
	}

	form := request.Form()
	if got := form.Get(yacyproto.FieldHashes); got != "Q_ylfl--9bK5eVZzCn--SAOx" {
		t.Errorf("hashes = %q, want the two names one after the other", got)
	}
	if got := form.Get(yacyproto.FieldCall); got != yacyproto.CallURLHashList {
		t.Errorf("call = %q, want %q", got, yacyproto.CallURLHashList)
	}
	if got := form.Get(yacyproto.FieldNetworkName); got != "freeworld" {
		t.Errorf("network name = %q, want %q", got, "freeworld")
	}
}

func TestURLMetadataResponseReadsARealYaCyAnswer(t *testing.T) {
	response, err := yacyproto.ParseURLMetadataResponse(
		context.Background(),
		[]byte(realYaCyURLMetadataBody),
	)
	if err != nil {
		t.Fatalf("ParseURLMetadataResponse: %v", err)
	}

	want := []yacymodel.URLMetadata{
		{
			Hash:     mustParseURLHash(t, "Q_ylfl--9bK5"),
			Address:  "https://www.ninjazumbi.com/",
			Title:    "NinjaZumbi's Dojo - A Blogh about anything",
			Author:   "Anarch157a",
			Modified: calendarDay(t, "2025-11-16"),
		},
		{
			Hash:     mustParseURLHash(t, "eVZzCn--SAOx"),
			Address:  "https://userbenchmarksweb.neocities.org/",
			Title:    "Userbenchmark's world",
			Modified: calendarDay(t, "2022-05-05"),
		},
	}
	if len(response.URLs) != len(want) {
		t.Fatalf("urls = %d, want %d", len(response.URLs), len(want))
	}
	for i, got := range response.URLs {
		if got.Hash != want[i].Hash || got.Address != want[i].Address ||
			got.Title != want[i].Title || got.Author != want[i].Author ||
			got.Modified != want[i].Modified {
			t.Errorf("url %d = %+v, want %+v", i, got, want[i])
		}
	}
}

func TestURLMetadataResponseDiscardsAnItemWithoutAName(t *testing.T) {
	const body = `<rss><yacy><response>ok</response></yacy><channel>` +
		`<item><link>http://example.com/</link><guid isPermaLink="false"></guid></item>` +
		`<item><link>http://example.com/x</link><guid isPermaLink="false">Q_ylfl--9bK5</guid></item>` +
		`</channel></rss>`

	response, err := yacyproto.ParseURLMetadataResponse(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("ParseURLMetadataResponse: %v", err)
	}
	if len(response.URLs) != 1 {
		t.Fatalf("urls = %d, want only the one the peer named", len(response.URLs))
	}
	if response.URLs[0].Address != "http://example.com/x" {
		t.Errorf("url = %q, want the named one", response.URLs[0].Address)
	}
}

func TestURLMetadataResponseReadsATitleThePeerLeftUnescaped(t *testing.T) {
	const body = `<rss><yacy><response>ok</response></yacy><channel>` +
		`<item><title>Rock & Roll</title><link>http://example.com/x</link>` +
		`<author>Smith & Sons</author>` +
		`<guid isPermaLink="false">Q_ylfl--9bK5</guid></item>` +
		`</channel></rss>`

	response, err := yacyproto.ParseURLMetadataResponse(context.Background(), []byte(body))
	if err != nil {
		t.Fatalf("ParseURLMetadataResponse: %v", err)
	}
	if len(response.URLs) != 1 {
		t.Fatalf("urls = %d, want the one the peer named", len(response.URLs))
	}
	if response.URLs[0].Title != "Rock & Roll" {
		t.Errorf("title = %q, want the ampersand as it stands", response.URLs[0].Title)
	}
	if response.URLs[0].Author != "Smith & Sons" {
		t.Errorf("author = %q, want the ampersand as it stands", response.URLs[0].Author)
	}
}

func TestURLMetadataResponseRefusesAPeerThatRejectedTheCall(t *testing.T) {
	const body = `<rss><yacy><response>rejected - insufficient call parameters</response></yacy>` +
		`<channel></channel></rss>`

	_, err := yacyproto.ParseURLMetadataResponse(context.Background(), []byte(body))
	if !errors.Is(err, yacyproto.ErrBadField) {
		t.Errorf("ParseURLMetadataResponse = %v, want ErrBadField", err)
	}
}

func calendarDay(t *testing.T, day string) yacymodel.Optional[yacymodel.CalendarDay] {
	t.Helper()

	instant, err := time.Parse(time.DateOnly, day)
	if err != nil {
		t.Fatalf("parse %q: %v", day, err)
	}

	return yacymodel.Some(yacymodel.CalendarDayOf(instant))
}
