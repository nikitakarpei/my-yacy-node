package main

import (
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

type pageFeedPreset struct {
	representation yacycrawlcontract.PageRepresentationKind
	enabled        bool
	rendering      crawlcapability.PageRendering
	build          func(jetstream.JetStream, string) crawlcapability.PageFeed
}

func pageFeedCatalog() []pageFeedPreset {
	text := pagetext.New()
	markdown := pagemarkdown.New()
	return []pageFeedPreset{
		{
			representation: yacycrawlcontract.PageRepresentationKindRWI,
			enabled:        true,
			rendering:      text,
			build: func(js jetstream.JetStream, subject string) crawlcapability.PageFeed {
				return pagefeed.NewRWIFeed(js, subject, text.Format())
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindText,
			enabled:        false,
			rendering:      text,
			build: func(js jetstream.JetStream, subject string) crawlcapability.PageFeed {
				return pagefeed.NewTextFeed(js, subject, text.Format())
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindMarkdown,
			enabled:        false,
			rendering:      markdown,
			build: func(js jetstream.JetStream, subject string) crawlcapability.PageFeed {
				return pagefeed.NewMarkdownFeed(js, subject, markdown.Format())
			},
		},
	}
}
