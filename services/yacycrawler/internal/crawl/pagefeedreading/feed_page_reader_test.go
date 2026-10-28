package pagefeedreading_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagefeedreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
)

const (
	feedAddress = "https://blog.example/feed"

	rssFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>Blog</title>
    <link>https://blog.example/</link>
    <atom:link href="https://blog.example/feed" rel="self"/>
    <item><title>One</title><link>https://blog.example/one</link><guid>https://blog.example/?p=1</guid></item>
    <item><title>Two</title><link>/two</link></item>
    <item><title>Three</title><guid>https://blog.example/three</guid></item>
    <item><title>Four</title><guid isPermaLink="false">https://blog.example/?p=4</guid></item>
    <item><title>Five</title><guid>urn:uuid:5</guid></item>
    <item><title>Broken</title><guid>http://[broken</guid></item>
    <item><title>Six</title><link>https://blog.example/one</link></item>
  </channel>
</rss>`

	rdfFeed = `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns="http://purl.org/rss/1.0/">
  <channel rdf:about="https://journal.example/rss">
    <title>Journal</title>
    <link>https://journal.example/</link>
    <items><rdf:Seq><rdf:li rdf:resource="https://journal.example/a1"/></rdf:Seq></items>
  </channel>
  <item rdf:about="https://journal.example/a1"><title>A1</title><link>https://journal.example/a1</link></item>
  <item rdf:about="https://journal.example/a2"><title>A2</title><link>https://journal.example/a2</link></item>
</rdf:RDF>`

	atomFeed = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xml:base="https://blog.example/posts/">
  <title>Blog</title>
  <link rel="self" href="/feed"/>
  <link href="https://blog.example/"/>
  <entry>
    <title>Hello</title>
    <link rel="alternate" href="hello-world"/>
    <link rel="replies" href="hello-world/comments"/>
    <link rel="enclosure" href="hello.mp3"/>
  </entry>
  <entry xml:base="/archive/">
    <title>Old</title>
    <link href="old-post"/>
    <link href="moved" xml:base="https://mirror.example/"/>
    <link href=""/>
  </entry>
</feed>`
)

func contentTypeOf(t *testing.T, declared string) pagereading.ContentType {
	t.Helper()
	contentType, err := pagereading.ContentTypeOf(declared)
	if err != nil {
		t.Fatalf("ContentTypeOf %q: %v", declared, err)
	}
	return contentType
}

func readingOfBody(
	t *testing.T,
	observer pagefeedreading.FeedReadingObserver,
	declaredContentType string,
	body []byte,
) (pagereading.BodyReading, bool) {
	t.Helper()
	reading, inPageFormat, err := pagefeedreading.NewFeedPageReader(
		linkresolution.New(silentLinkResolutionObserver{}), observer,
	).ReadingOfBody(
		t.Context(),
		canonicalurltest.CanonicalURLOf(t, feedAddress),
		contentTypeOf(t, declaredContentType),
		body,
	)
	if err != nil {
		t.Fatalf("ReadingOfBody: %v", err)
	}
	return reading, inPageFormat
}

func readingOfFeed(t *testing.T, declaredContentType, feed string) pagereading.BodyReading {
	t.Helper()
	observer := &recordingFeedReadingObserver{}
	reading, inPageFormat := readingOfBody(t, observer, declaredContentType, []byte(feed))
	if !inPageFormat {
		t.Fatal("want the body read as a feed")
	}
	if observer.unreadableFeeds != 0 {
		t.Fatalf("unreadable feeds = %d, want 0", observer.unreadableFeeds)
	}
	return reading
}

func TestReadingOfBodyReportsTheSiteAndTheItemsOfAnRSSFeed(t *testing.T) {
	reading := readingOfFeed(t, "application/rss+xml", rssFeed)

	assertURLs(t, reading.LinkedURLs,
		"https://blog.example/",
		"https://blog.example/one",
		"https://blog.example/two",
		"https://blog.example/three",
	)
}

func TestReadingOfBodyReportsTheSiteAndTheItemsOfAnRSS1Feed(t *testing.T) {
	reading := readingOfFeed(t, "application/rdf+xml", rdfFeed)

	assertURLs(t, reading.LinkedURLs,
		"https://journal.example/",
		"https://journal.example/a1",
		"https://journal.example/a2",
	)
}

func TestReadingOfBodyReportsTheAlternateLinksOfAnAtomFeedAgainstTheirBaseURLs(t *testing.T) {
	reading := readingOfFeed(t, "application/atom+xml", atomFeed)

	assertURLs(t, reading.LinkedURLs,
		"https://blog.example/",
		"https://blog.example/posts/hello-world",
		"https://blog.example/archive/old-post",
		"https://mirror.example/moved",
	)
}

func TestReadingOfBodyReadsAFeedServedAsPlainXML(t *testing.T) {
	for _, declaredContentType := range []string{"text/xml", "application/xml; charset=utf-8"} {
		reading := readingOfFeed(t, declaredContentType, atomFeed)

		if len(reading.LinkedURLs) != 4 {
			t.Errorf("%s: links = %v", declaredContentType, reading.LinkedURLs)
		}
	}
}

func TestReadingOfBodyLeavesABodyThatIsNotAFeedUnread(t *testing.T) {
	for name, test := range map[string]struct {
		contentType string
		body        string
	}{
		"html":          {contentType: "text/html", body: rssFeed},
		"other xml":     {contentType: "image/svg+xml", body: `<svg xmlns="http://www.w3.org/2000/svg"/>`},
		"sitemap":       {contentType: "text/xml", body: `<urlset><url><loc>https://a.example/</loc></url></urlset>`},
		"broken xml":    {contentType: "text/xml", body: `not xml at all`},
		"atom 0.3 feed": {contentType: "text/xml", body: `<feed xmlns="http://purl.org/atom/ns#"/>`},
	} {
		t.Run(name, func(t *testing.T) {
			_, inPageFormat := readingOfBody(
				t, &recordingFeedReadingObserver{}, test.contentType, []byte(test.body),
			)

			if inPageFormat {
				t.Fatal("want the body left to another format")
			}
		})
	}
}

func TestReadingOfBodyFollowsTheLinksReadBeforeAFeedBreaksOff(t *testing.T) {
	observer := &recordingFeedReadingObserver{}
	brokenFeed := `<rss><channel><item><link>https://blog.example/one</link></item>` +
		`<item><link>https://blog.example/two</link></item><item><link>https://blog`

	reading, inPageFormat := readingOfBody(t, observer, "application/rss+xml", []byte(brokenFeed))

	if !inPageFormat {
		t.Fatal("want the body read as a feed")
	}
	assertURLs(t, reading.LinkedURLs, "https://blog.example/one", "https://blog.example/two")
	if observer.unreadableFeeds != 1 {
		t.Fatalf("unreadable feeds = %d, want 1", observer.unreadableFeeds)
	}
}

func TestReadingOfBodyReadsTheHTMLEntitiesAFeedHolds(t *testing.T) {
	reading := readingOfFeed(
		t,
		"application/rss+xml",
		`<rss><channel><item><title>A&nbsp;B</title><link>https://blog.example/a?x=1&amp;y=2</link></item></channel></rss>`,
	)

	assertURLs(t, reading.LinkedURLs, "https://blog.example/a?x=1&y=2")
}

func TestReadingOfBodyDecodesTheCharsetTheFeedDeclares(t *testing.T) {
	feed := []byte(`<?xml version="1.0" encoding="ISO-8859-1"?>` +
		"<rss><channel><item><link>https://blog.example/caf\xe9</link></item></channel></rss>")

	reading := readingOfFeed(t, "application/rss+xml", string(feed))

	assertURLs(t, reading.LinkedURLs, "https://blog.example/caf%C3%A9")
}

func TestReadingOfBodyPrefersTheCharsetOfTheContentType(t *testing.T) {
	feed := []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
		"<rss><channel><item><link>https://blog.example/caf\xe9</link></item></channel></rss>")

	reading := readingOfFeed(t, "text/xml; charset=ISO-8859-1", string(feed))

	assertURLs(t, reading.LinkedURLs, "https://blog.example/caf%C3%A9")
}

func TestReadingOfBodyDecodesTheCharsetOfTheContentTypeOnce(t *testing.T) {
	feed := []byte(`<?xml version="1.0" encoding="ISO-8859-1"?>` +
		"<rss><channel><item><link>https://blog.example/caf\xe9</link></item></channel></rss>")

	reading := readingOfFeed(t, "text/xml; charset=ISO-8859-1", string(feed))

	assertURLs(t, reading.LinkedURLs, "https://blog.example/caf%C3%A9")
}

func assertURLs(t *testing.T, got []canonicalurl.CanonicalURL, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("urls = %v, want %v", got, want)
	}
	for i, wanted := range want {
		if got[i].String() != wanted {
			t.Fatalf("urls = %v, want %v", got, want)
		}
	}
}

type recordingFeedReadingObserver struct {
	unreadableFeeds int
}

func (o *recordingFeedReadingObserver) FeedUnreadable(
	context.Context, canonicalurl.CanonicalURL, error,
) {
	o.unreadableFeeds++
}

type silentLinkResolutionObserver struct{}

func (silentLinkResolutionObserver) BaseURLUnresolved(
	context.Context, canonicalurl.CanonicalURL, string, error,
) {
}

func (silentLinkResolutionObserver) LinksUnresolved(
	context.Context, canonicalurl.CanonicalURL, int,
) {
}
