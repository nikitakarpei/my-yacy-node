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
	queryWords []yacymodel.Hash,
) networksearch.PageReadingRun {
	return pageReading.reading.Start(queryWords)
}

func (pageReading networkSearchPageReading) ReadPagesAmong(
	ctx context.Context,
	run networksearch.PageReadingRun,
	pagesWanted []pagereading.PageToRead,
) pagereading.ReadPages {
	//nolint:forcetypeassert // every run this reading gets back is one its Start gave out
	return pageReading.reading.ReadPagesAmong(ctx, run.(*pagereading.Run), pagesWanted)
}
