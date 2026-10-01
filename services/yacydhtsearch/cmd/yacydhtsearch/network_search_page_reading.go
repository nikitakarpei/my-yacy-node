package main

import (
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
