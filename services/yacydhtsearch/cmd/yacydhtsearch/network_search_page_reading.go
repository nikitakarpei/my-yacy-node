package main

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type networkSearchPageReading struct {
	reading pagereading.Reading
}

func (pageReading networkSearchPageReading) Start(
	ctx context.Context,
	queryWords []yacymodel.Hash,
) networksearch.PageReadingRun {
	return pageReading.reading.Start(ctx, queryWords)
}
