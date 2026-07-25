package main

import (
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerepresentations/rwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagerepresentations/text"
	representationpublishersjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/representationpublishers/jetstream"
)

type pageRepresentationPreset struct {
	representation yacycrawlcontract.PageRepresentationKind
	enabled        bool
	build          func(jetstream.JetStream, string) pagepublication.PageRepresentation
}

func pageRepresentationCatalog() []pageRepresentationPreset {
	return []pageRepresentationPreset{
		{
			representation: yacycrawlcontract.PageRepresentationKindRWI,
			enabled:        true,
			build: func(js jetstream.JetStream, subject string) pagepublication.PageRepresentation {
				return representationpublishersjetstream.Wrap(rwi.New(), subject, js)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindText,
			enabled:        false,
			build: func(js jetstream.JetStream, subject string) pagepublication.PageRepresentation {
				return representationpublishersjetstream.Wrap(text.New(), subject, js)
			},
		},
		{
			representation: yacycrawlcontract.PageRepresentationKindMarkdown,
			enabled:        false,
			build: func(js jetstream.JetStream, subject string) pagepublication.PageRepresentation {
				return representationpublishersjetstream.Wrap(markdown.New(), subject, js)
			},
		},
	}
}
