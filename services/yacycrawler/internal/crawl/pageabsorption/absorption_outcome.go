package pageabsorption

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type AbsorptionOutcome struct {
	DiscoveredURLs []yacycrawlcontract.CanonicalURL
	Disposal       disposal.Reason
}

type absorbedDocument struct {
	discoveredURLs []yacycrawlcontract.CanonicalURL
	disposal       disposal.Reason
}

func absorptionOutcomeFrom(documents []absorbedDocument) AbsorptionOutcome {
	outcome := AbsorptionOutcome{Disposal: pageDisposalFrom(documents)}
	for _, document := range documents {
		outcome.DiscoveredURLs = append(outcome.DiscoveredURLs, document.discoveredURLs...)
	}
	return outcome
}

func pageDisposalFrom(documents []absorbedDocument) disposal.Reason {
	for _, document := range documents {
		if document.disposal == disposal.NotDisposed {
			return disposal.NotDisposed
		}
	}
	return documents[0].disposal
}
