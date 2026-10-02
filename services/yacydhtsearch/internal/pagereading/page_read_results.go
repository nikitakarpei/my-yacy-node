package pagereading

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type pageReadResults []pageReadResult

func (results pageReadResults) withUnsettledPagesCutOff(pagesWanted []PageToRead) pageReadResults {
	for _, unsettledPage := range results.pagesUnsettledAmong(pagesWanted) {
		results = append(results, pageReadResult{
			document: unsettledPage.Document,
			outcome:  pageWasCutOff,
		})
	}

	return results
}

func (results pageReadResults) pagesUnsettledAmong(pagesWanted []PageToRead) []PageToRead {
	settledDocuments := make(map[yacymodel.URLHash]struct{}, len(results))
	for _, result := range results {
		settledDocuments[result.document] = struct{}{}
	}
	pagesUnsettled := make([]PageToRead, 0, len(pagesWanted))
	for _, pageWanted := range pagesWanted {
		if _, settled := settledDocuments[pageWanted.Document]; !settled {
			pagesUnsettled = append(pagesUnsettled, pageWanted)
		}
	}

	return pagesUnsettled
}

func (results pageReadResults) withUnsettledPagesOutOfBudget(
	pagesWanted []PageToRead,
) pageReadResults {
	for _, unsettledPage := range results.pagesUnsettledAmong(pagesWanted) {
		results = append(results, pageReadResult{
			document: unsettledPage.Document,
			outcome:  pageWasOutOfBudget,
		})
	}

	return results
}
