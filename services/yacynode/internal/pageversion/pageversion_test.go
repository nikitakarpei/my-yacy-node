package pageversion_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const storedDay yacymodel.MicroDate = 20000

var (
	sendingNetwork = rwiorigin.Of("203.0.113.7")
	otherNetwork   = rwiorigin.Of("198.51.100.9")
	crawlDay       = yacymodel.NewCalendarDay(2020, time.January, 1)
)

func versionOn(day yacymodel.MicroDate, titleWords uint8) pageversion.Version {
	return pageversion.FromPosting(yacymodel.RWIPosting{
		LastModified: day,
		TitleWords:   titleWords,
	}, pageversion.SourcePeer(sendingNetwork))
}

func TestArrivedVersionIsTheStoredOneFromAnySource(t *testing.T) {
	stored := versionOn(storedDay, 2)
	samePosting := yacymodel.RWIPosting{LastModified: storedDay, TitleWords: 2}
	for name, c := range map[string]struct {
		arrived pageversion.Version
		want    bool
	}{
		"same day, same fingerprint":  {versionOn(storedDay, 2), true},
		"same day, other fingerprint": {versionOn(storedDay, 3), false},
		"older day":                   {versionOn(storedDay-1, 2), false},
		"newer day":                   {versionOn(storedDay+1, 2), false},
		"same fingerprint from another network": {
			pageversion.FromPosting(samePosting, pageversion.SourcePeer(otherNetwork)), true,
		},
		"same fingerprint from this node": {
			pageversion.FromPosting(samePosting, pageversion.SourceThisNode()), true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := stored.Is(c.arrived); got != c.want {
				t.Fatalf("Is = %v, want %v", got, c.want)
			}
		})
	}
}

func TestArrivedVersionReplacesTheStoredOneOnlyOnANewerDay(t *testing.T) {
	stored := versionOn(storedDay, 2)
	for name, c := range map[string]struct {
		arrived pageversion.Version
		want    bool
	}{
		"older day":                    {versionOn(storedDay-1, 2), false},
		"same day, same fingerprint":   {versionOn(storedDay, 2), false},
		"same day, other fingerprint":  {versionOn(storedDay, 3), false},
		"newer day":                    {versionOn(storedDay+1, 2), true},
		"newer day, other fingerprint": {versionOn(storedDay+1, 3), true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := stored.IsReplacedBy(c.arrived); got != c.want {
				t.Fatalf("IsReplacedBy = %v, want %v", got, c.want)
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
				pageversion.SourcePeer(sendingNetwork),
			)
			if got := version.IsDatedAfter(lateOnCrawlDay); got != c.want {
				t.Fatalf("IsDatedAfter = %v, want %v", got, c.want)
			}
		})
	}
}

func TestArrivedVersionJoinsOnlyTheSameVersionFromTheSameSource(t *testing.T) {
	modified, inRange := yacymodel.MicroDateOf(crawlDay.Time()).Get()
	if !inRange {
		t.Fatalf("MicroDateOf(%v) is absent", crawlDay)
	}
	metadata := yacymodel.URLMetadata{
		Title:        "Hello World",
		Modified:     yacymodel.Some(crawlDay),
		DocumentType: yacymodel.DocumentTypeText,
	}
	postingOfVersion := yacymodel.RWIPosting{
		LastModified: modified,
		TitleWords:   2,
		Language:     yacymodel.LanguageOfUndeclaredDocument,
		DocumentType: yacymodel.DocumentTypeText,
	}
	postingOfOtherVersion := postingOfVersion
	postingOfOtherVersion.TitleWords = 3
	fromSendingNetwork := pageversion.SourcePeer(sendingNetwork)
	fromOtherNetwork := pageversion.SourcePeer(otherNetwork)
	sentByPeer := pageversion.FromMetadata(metadata, fromSendingNetwork)
	crawledHere := pageversion.FromMetadata(metadata, pageversion.SourceThisNode())
	for name, c := range map[string]struct {
		stored  pageversion.Version
		arrived pageversion.Version
		want    pageversion.Joining
	}{
		"same version, same source": {
			sentByPeer,
			pageversion.FromPosting(postingOfVersion, fromSendingNetwork),
			pageversion.JoiningAllowed,
		},
		"same version, other network": {
			sentByPeer,
			pageversion.FromPosting(postingOfVersion, fromOtherNetwork),
			pageversion.JoiningRefusedOtherSource,
		},
		"same version, crawled here": {
			crawledHere,
			pageversion.FromPosting(postingOfVersion, fromSendingNetwork),
			pageversion.JoiningRefusedOtherSource,
		},
		"other version, same source": {
			sentByPeer,
			pageversion.FromPosting(postingOfOtherVersion, fromSendingNetwork),
			pageversion.JoiningRefusedOtherVersion,
		},
		"other version, other network": {
			sentByPeer,
			pageversion.FromPosting(postingOfOtherVersion, fromOtherNetwork),
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
