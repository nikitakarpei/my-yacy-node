package networksearch

import (
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

func documentsToReadAmong(
	orderedDocuments []queryanswers.FoundDocument,
	pagesReadPerQuery int,
	pagesReadPerSite int,
) []queryanswers.FoundDocument {
	documentsToRead := make([]queryanswers.FoundDocument, 0, pagesReadPerQuery)
	amountOfPagesPerSite := map[string]int{}
	for _, document := range orderedDocuments {
		if len(documentsToRead) >= pagesReadPerQuery {
			break
		}
		site := siteOf(document.Address)
		if amountOfPagesPerSite[site] >= pagesReadPerSite {
			continue
		}
		amountOfPagesPerSite[site]++
		documentsToRead = append(documentsToRead, document)
	}

	return documentsToRead
}

func siteOf(address string) string {
	webAddress, err := url.Parse(address)
	if err != nil {
		return ""
	}

	return webAddress.Hostname()
}
