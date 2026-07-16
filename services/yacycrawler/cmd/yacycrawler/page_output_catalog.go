package main

import (
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagemarkdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagetext"
)

type pageOutputPreset struct {
	representation yacycrawlcontract.PageRepresentation
	subject        string
	enabled        bool
	build          func(jetstream.JetStream, string) crawlcapability.PageRepresentationOutput
}

func pageOutputCatalog() []pageOutputPreset {
	text := pagetext.New()
	markdown := pagemarkdown.New()
	return []pageOutputPreset{
		{
			representation: yacycrawlcontract.PageRepresentationRWI,
			subject:        "yacy.crawl.page.rwi",
			enabled:        true,
			build: func(
				js jetstream.JetStream,
				subject string,
			) crawlcapability.PageRepresentationOutput {
				return crawlcapability.BindRepresentation(
					yacycrawlcontract.PageRepresentationRWI,
					pagerwi.NewDerivation(text),
					pagepublication.NewRWIPublication(js, subject),
				)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationText,
			subject:        "yacy.crawl.page.text",
			enabled:        false,
			build: func(
				js jetstream.JetStream,
				subject string,
			) crawlcapability.PageRepresentationOutput {
				return crawlcapability.BindRepresentation(
					yacycrawlcontract.PageRepresentationText,
					pagetext.NewDerivation(text),
					pagepublication.NewTextPublication(js, subject),
				)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationMarkdown,
			subject:        "yacy.crawl.page.markdown",
			enabled:        false,
			build: func(
				js jetstream.JetStream,
				subject string,
			) crawlcapability.PageRepresentationOutput {
				return crawlcapability.BindRepresentation(
					yacycrawlcontract.PageRepresentationMarkdown,
					pagemarkdown.NewDerivation(markdown),
					pagepublication.NewMarkdownPublication(js, subject),
				)
			},
		},
	}
}
