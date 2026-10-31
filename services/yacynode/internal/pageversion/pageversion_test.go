package pageversion_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const storedDay yacymodel.MicroDate = 20000

const crawlerRingExponent = 2

var (
	sendingNetwork = rwiorigin.Of("203.0.113.7")
	otherNetwork   = rwiorigin.Of("198.51.100.9")
	crawlDay       = yacymodel.NewCalendarDay(2020, time.January, 1)
	fromSender     = pageversion.SourcePeer(sendingNetwork)
	fromOther      = pageversion.SourcePeer(otherNetwork)
)

func postingOn(day yacymodel.MicroDate, titleWords uint8) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:     yacymodel.WordHash("word"),
		LastModified: day,
		TitleWords:   titleWords,
	}
}

func unrootedVersionOf(
	posting yacymodel.RWIPosting,
	source pageversion.Source,
) pageversion.Version {
	return pageversion.FromPostingAlone(yacymodel.UnprovenPostingAloneFrom(posting), source)
}

func rootedVersionOf(
	posting yacymodel.RWIPosting,
	source pageversion.Source,
) pageversion.Version {
	return pageversion.FromPostingAlone(crawledPostingsAloneFrom(posting)[0], source)
}

func crawledPostingsAloneFrom(postings ...yacymodel.RWIPosting) []yacymodel.PostingAlone {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(crawlerRingExponent)
	if err != nil {
		panic(err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		postings,
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		panic(err)
	}

	return crawled.EachPostingAlone()
}

func versionUnderAnotherRoot(
	posting yacymodel.RWIPosting,
	source pageversion.Source,
) pageversion.Version {
	otherWord := posting
	otherWord.WordHash = yacymodel.WordHash("other word")

	return pageversion.FromPostingAlone(crawledPostingsAloneFrom(posting, otherWord)[0], source)
}

func TestArrivedVersionIsTheStoredOneFromAnySource(t *testing.T) {
	stored := unrootedVersionOf(postingOn(storedDay, 2), fromSender)
	rootedStored := rootedVersionOf(postingOn(storedDay, 2), fromSender)
	for name, c := range map[string]struct {
		stored  pageversion.Version
		arrived pageversion.Version
		want    bool
	}{
		"same day, same fingerprint": {
			stored, unrootedVersionOf(postingOn(storedDay, 2), fromSender), true,
		},
		"same day, other fingerprint": {
			stored, unrootedVersionOf(postingOn(storedDay, 3), fromSender), false,
		},
		"older day": {stored, unrootedVersionOf(postingOn(storedDay-1, 2), fromSender), false},
		"newer day": {stored, unrootedVersionOf(postingOn(storedDay+1, 2), fromSender), false},
		"same fingerprint from another network": {
			stored, unrootedVersionOf(postingOn(storedDay, 2), fromOther), true,
		},
		"same fingerprint from this node": {
			stored, unrootedVersionOf(postingOn(storedDay, 2), pageversion.SourceThisNode()), true,
		},
		"same fingerprint with a root": {
			stored, rootedVersionOf(postingOn(storedDay, 2), fromSender), false,
		},
		"same root from another network": {
			rootedStored, rootedVersionOf(postingOn(storedDay, 2), fromOther), true,
		},
		"another root": {
			rootedStored, versionUnderAnotherRoot(postingOn(storedDay, 2), fromSender), false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := c.stored.Is(c.arrived); got != c.want {
				t.Fatalf("Is = %v, want %v", got, c.want)
			}
		})
	}
}

func TestArrivedVersionReplacesTheStoredOneOnANewerDayOrWithTheFirstRootOfTheDay(t *testing.T) {
	stored := unrootedVersionOf(postingOn(storedDay, 2), fromSender)
	rootedStored := rootedVersionOf(postingOn(storedDay, 2), fromSender)
	for name, c := range map[string]struct {
		stored  pageversion.Version
		arrived pageversion.Version
		want    pageversion.Replacement
	}{
		"older day": {
			stored,
			unrootedVersionOf(postingOn(storedDay-1, 2), fromSender),
			pageversion.ReplacementRefused,
		},
		"same day, same fingerprint": {
			stored,
			unrootedVersionOf(postingOn(storedDay, 2), fromSender),
			pageversion.ReplacementRefused,
		},
		"same day, other fingerprint": {
			stored,
			unrootedVersionOf(postingOn(storedDay, 3), fromSender),
			pageversion.ReplacementRefused,
		},
		"newer day": {
			stored,
			unrootedVersionOf(postingOn(storedDay+1, 2), fromSender),
			pageversion.ReplacementAllowedOlderVersion,
		},
		"newer day, other fingerprint": {
			stored,
			unrootedVersionOf(postingOn(storedDay+1, 3), fromSender),
			pageversion.ReplacementAllowedOlderVersion,
		},
		"same day with a root": {
			stored,
			rootedVersionOf(postingOn(storedDay, 2), fromOther),
			pageversion.ReplacementAllowedUnrootedVersion,
		},
		"same day, other fingerprint with a root": {
			stored,
			rootedVersionOf(postingOn(storedDay, 3), fromSender),
			pageversion.ReplacementAllowedUnrootedVersion,
		},
		"older day with a root": {
			stored,
			rootedVersionOf(postingOn(storedDay-1, 2), fromSender),
			pageversion.ReplacementRefused,
		},
		"same day, another root": {
			rootedStored,
			versionUnderAnotherRoot(postingOn(storedDay, 2), fromSender),
			pageversion.ReplacementRefused,
		},
		"same day without a root": {
			rootedStored,
			unrootedVersionOf(postingOn(storedDay, 2), fromSender),
			pageversion.ReplacementRefused,
		},
		"newer day without a root": {
			rootedStored,
			unrootedVersionOf(postingOn(storedDay+1, 2), fromSender),
			pageversion.ReplacementAllowedOlderVersion,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := c.stored.ReplacementBy(c.arrived); got != c.want {
				t.Fatalf("ReplacementBy = %v, want %v", got, c.want)
			}
			if got := c.stored.IsReplacedBy(
				c.arrived,
			); got != (c.want != pageversion.ReplacementRefused) {
				t.Fatalf("IsReplacedBy = %v, want it for %v", got, c.want)
			}
		})
	}
}

func TestVersionIsDatedAfterNowOnlyOnALaterDay(t *testing.T) {
	lateOnCrawlDay := crawlDay.Time().Add(23 * time.Hour)
	for name, c := range map[string]struct {
		modified yacymodel.CalendarDay
		want     bool
	}{
		"same day":           {yacymodel.NewCalendarDay(2020, time.January, 1), false},
		"day before":         {yacymodel.NewCalendarDay(2019, time.December, 31), false},
		"next day":           {yacymodel.NewCalendarDay(2020, time.January, 2), true},
		"before the epoch":   {yacymodel.NewCalendarDay(1965, time.March, 4), false},
		"past the two bytes": {yacymodel.NewCalendarDay(2200, time.May, 6), false},
	} {
		t.Run(name, func(t *testing.T) {
			version := pageversion.FromMetadata(
				yacymodel.URLMetadata{Modified: yacymodel.Some(c.modified)},
				fromSender,
			)
			if got := version.IsDatedAfter(lateOnCrawlDay); got != c.want {
				t.Fatalf("IsDatedAfter = %v, want %v", got, c.want)
			}
		})
	}
}

type crawlOnCrawlDay struct {
	postingOfVersion      yacymodel.RWIPosting
	postingOfOtherVersion yacymodel.RWIPosting
	metadata              yacymodel.URLMetadata
	rootedMetadata        yacymodel.URLMetadata
}

func crawlOnCrawlDayOf(t *testing.T) crawlOnCrawlDay {
	t.Helper()

	modified, inRange := yacymodel.MicroDateOf(crawlDay.Time()).Get()
	if !inRange {
		t.Fatalf("MicroDateOf(%v) is absent", crawlDay)
	}
	crawl := crawlOnCrawlDay{
		postingOfVersion: yacymodel.RWIPosting{
			WordHash:     yacymodel.WordHash("word"),
			LastModified: modified,
			TitleWords:   2,
			Language:     yacymodel.LanguageOfUndeclaredDocument,
			DocumentType: yacymodel.DocumentTypeText,
		},
		metadata: yacymodel.URLMetadata{
			Title:        "Hello World",
			Modified:     yacymodel.Some(crawlDay),
			DocumentType: yacymodel.DocumentTypeText,
		},
	}
	crawl.postingOfOtherVersion = crawl.postingOfVersion
	crawl.postingOfOtherVersion.TitleWords = 3
	crawl.rootedMetadata = crawl.metadata
	crawl.rootedMetadata.PostingsRoot = crawledPostingsAloneFrom(crawl.postingOfVersion)[0].Root()

	return crawl
}

func TestArrivedVersionJoinsAnUnrootedVersionOnlyFromTheSameSource(t *testing.T) {
	crawl := crawlOnCrawlDayOf(t)
	sentByPeer := pageversion.FromMetadata(crawl.metadata, fromSender)
	crawledHere := pageversion.FromMetadata(crawl.metadata, pageversion.SourceThisNode())
	for name, c := range map[string]struct {
		stored  pageversion.Version
		arrived pageversion.Version
		want    pageversion.Joining
	}{
		"same version, same source": {
			sentByPeer,
			unrootedVersionOf(crawl.postingOfVersion, fromSender),
			pageversion.JoiningAllowed,
		},
		"same version, other network": {
			sentByPeer,
			unrootedVersionOf(crawl.postingOfVersion, fromOther),
			pageversion.JoiningRefusedOtherSource,
		},
		"same version, crawled here": {
			crawledHere,
			unrootedVersionOf(crawl.postingOfVersion, fromSender),
			pageversion.JoiningRefusedOtherSource,
		},
		"other version, same source": {
			sentByPeer,
			unrootedVersionOf(crawl.postingOfOtherVersion, fromSender),
			pageversion.JoiningRefusedOtherVersion,
		},
		"other version, other network": {
			sentByPeer,
			unrootedVersionOf(crawl.postingOfOtherVersion, fromOther),
			pageversion.JoiningRefusedOtherVersion,
		},
		"rooted against an unrooted version": {
			sentByPeer,
			rootedVersionOf(crawl.postingOfVersion, fromSender),
			pageversion.JoiningRefusedOtherVersion,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := c.stored.JoiningOf(c.arrived); got != c.want {
				t.Fatalf("JoiningOf = %v, want %v", got, c.want)
			}
		})
	}
}

func TestArrivedVersionJoinsARootedVersionFromAnySourceOnlyUnderItsRoot(t *testing.T) {
	crawl := crawlOnCrawlDayOf(t)
	rootedByPeer := pageversion.FromMetadata(crawl.rootedMetadata, fromSender)
	rootedHere := pageversion.FromMetadata(crawl.rootedMetadata, pageversion.SourceThisNode())
	for name, c := range map[string]struct {
		stored  pageversion.Version
		arrived pageversion.Version
		want    pageversion.Joining
	}{
		"under the root, other network": {
			rootedByPeer,
			rootedVersionOf(crawl.postingOfVersion, fromOther),
			pageversion.JoiningAllowed,
		},
		"under the root, crawled here": {
			rootedHere,
			rootedVersionOf(crawl.postingOfVersion, fromSender),
			pageversion.JoiningAllowed,
		},
		"same fingerprint without a root": {
			rootedByPeer,
			unrootedVersionOf(crawl.postingOfVersion, fromSender),
			pageversion.JoiningRefusedUnrootedVersion,
		},
		"under another root": {
			rootedByPeer,
			versionUnderAnotherRoot(crawl.postingOfVersion, fromSender),
			pageversion.JoiningRefusedOtherVersion,
		},
		"other fingerprint without a root": {
			rootedByPeer,
			unrootedVersionOf(crawl.postingOfOtherVersion, fromSender),
			pageversion.JoiningRefusedOtherVersion,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := c.stored.JoiningOf(c.arrived); got != c.want {
				t.Fatalf("JoiningOf = %v, want %v", got, c.want)
			}
		})
	}
}

func TestVersionBytesTellVersionsApartButNotSources(t *testing.T) {
	posting := postingOn(storedDay, 2)
	unrooted := pageversion.AppendBytes(unrootedVersionOf(posting, fromSender), nil)
	rooted := pageversion.AppendBytes(rootedVersionOf(posting, fromSender), nil)

	if !bytes.Equal(unrooted, pageversion.AppendBytes(unrootedVersionOf(posting, fromOther), nil)) {
		t.Fatal("the source changes the bytes of a version")
	}
	if !bytes.Equal(
		unrooted,
		pagefingerprint.AppendBytes(pagefingerprint.FromPosting(posting), nil),
	) {
		t.Fatal("a version without a root has other bytes than its fingerprint")
	}
	if bytes.Equal(rooted, unrooted) ||
		bytes.Equal(
			rooted,
			pageversion.AppendBytes(versionUnderAnotherRoot(posting, fromSender), nil),
		) {
		t.Fatal("two roots of one fingerprint share their bytes")
	}
}
