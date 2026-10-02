package networksearch

import (
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

func documentsToReadAmong(
	orderedDocuments []queryfindings.FoundDocument,
	pagesReadPerQuery int,
	pagesReadPerSite int,
) []queryfindings.FoundDocument {
	documentsToRead := make([]queryfindings.FoundDocument, 0, pagesReadPerQuery)
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
