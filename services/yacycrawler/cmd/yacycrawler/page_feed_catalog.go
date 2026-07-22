package main

import (
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	feedpublishersjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/feedpublishers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeeds/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeeds/rwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeeds/text"
)

type pageFeedPreset struct {
	representation yacycrawlcontract.PageRepresentationKind
	enabled        bool
	build          func(jetstream.JetStream, string) pageabsorption.Feed
}

func pageFeedCatalog() []pageFeedPreset {
	return []pageFeedPreset{
		{
			representation: yacycrawlcontract.PageRepresentationKindRWI,
			enabled:        true,
			build: func(js jetstream.JetStream, subject string) pageabsorption.Feed {
				return feedpublishersjetstream.Wrap(rwi.New(), subject, js)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindText,
			enabled:        false,
			build: func(js jetstream.JetStream, subject string) pageabsorption.Feed {
				return feedpublishersjetstream.Wrap(text.New(), subject, js)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindMarkdown,
			enabled:        false,
			build: func(js jetstream.JetStream, subject string) pageabsorption.Feed {
				return feedpublishersjetstream.Wrap(markdown.New(), subject, js)
			},
		},
	}
}
