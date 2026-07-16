package main

import (
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

type pageFeedPreset struct {
	representation yacycrawlcontract.PageRepresentationKind
	enabled        bool
	build          func(jetstream.JetStream, string) crawlcapability.PageFeed
}

func pageFeedCatalog() []pageFeedPreset {
	text := pagetext.New()
	markdown := pagemarkdown.New()
	return []pageFeedPreset{
		{
			representation: yacycrawlcontract.PageRepresentationKindRWI,
			enabled:        true,
			build: func(
				js jetstream.JetStream,
				subject string,
			) crawlcapability.PageFeed {
				return pagefeed.Bind(
					yacycrawlcontract.PageRepresentationKindRWI,
					pagerwi.NewDerivation(text),
					pagepublication.NewRWIPublication(js, subject),
				)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindText,
			enabled:        false,
			build: func(
				js jetstream.JetStream,
				subject string,
			) crawlcapability.PageFeed {
				return pagefeed.Bind(
					yacycrawlcontract.PageRepresentationKindText,
					pagetext.NewDerivation(text),
					pagepublication.NewTextPublication(js, subject),
				)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindMarkdown,
			enabled:        false,
			build: func(
				js jetstream.JetStream,
				subject string,
			) crawlcapability.PageFeed {
				return pagefeed.Bind(
					yacycrawlcontract.PageRepresentationKindMarkdown,
					pagemarkdown.NewDerivation(markdown),
					pagepublication.NewMarkdownPublication(js, subject),
				)
			},
		},
	}
}
