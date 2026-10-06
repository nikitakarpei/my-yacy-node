package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
)

type networkSearchPageReading struct {
	reading pagereading.Reading
}

func (pageReading networkSearchPageReading) Start(
	queryWords []string,
) networksearch.PageReadingRun {
	return pageReading.reading.Start(queryWords)
}
