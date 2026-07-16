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
	build          func(jetstream.JetStream, string) crawlcapability.PageFeed
}

func pageFeedCatalog() []pageFeedPreset {
	return []pageFeedPreset{
		{
			representation: yacycrawlcontract.PageRepresentationKindRWI,
			enabled:        true,
			build: func(js jetstream.JetStream, subject string) crawlcapability.PageFeed {
				return pagefeed.NewRWIFeed(js, subject)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindText,
			enabled:        false,
			build: func(js jetstream.JetStream, subject string) crawlcapability.PageFeed {
				return pagefeed.NewTextFeed(js, subject)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindMarkdown,
			enabled:        false,
			build: func(js jetstream.JetStream, subject string) crawlcapability.PageFeed {
				return pagefeed.NewMarkdownFeed(js, subject)
			},
		},
	}
}

func pageRenderingCatalog() []crawlcapability.PageRendering {
	return []crawlcapability.PageRendering{pagetext.New(), pagemarkdown.New()}
}
