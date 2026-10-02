package networksearch

import (
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

func pagesToReadAmong(
	orderedDocuments []queryfindings.FoundDocument,
	pagesReadPerQuery int,
	pagesReadPerSite int,
) []pagereading.PageToRead {
	pagesToRead := make([]pagereading.PageToRead, 0, pagesReadPerQuery)
	amountOfPagesPerSite := map[string]int{}
	for _, document := range orderedDocuments {
		if len(pagesToRead) >= pagesReadPerQuery {
			break
		}
		site := siteOf(document.Address)
		if amountOfPagesPerSite[site] >= pagesReadPerSite {
			continue
		}
		amountOfPagesPerSite[site]++
		pagesToRead = append(pagesToRead, pagereading.PageToRead{
			Document: document.Hash,
			Address:  document.Address,
		})
	}

	return pagesToRead
}

func siteOf(address string) string {
	webAddress, err := url.Parse(address)
	if err != nil {
		return ""
	}

	return webAddress.Hostname()
}
