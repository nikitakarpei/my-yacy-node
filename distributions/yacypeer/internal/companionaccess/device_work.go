package companionaccess

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	jetStreamInfo     = "$JS.API.INFO"
	streamInfoStart   = "$JS.API.STREAM.INFO."
	consumerInfoStart = "$JS.API.CONSUMER.INFO."
	consumerListStart = "$JS.API.CONSUMER.LIST."
	nextMessageStart  = "$JS.API.CONSUMER.MSG.NEXT."
	ackStart          = "$JS.ACK."
)

func DeviceWorkFor(scrapeRequestDurable string, pageOfferDurable string) WorkSubjects {
	scrapeRequests := pagescrapecontract.ScrapeRequestsStreamName
	pageOffers := pagescrapecontract.ScrapePageOffersStreamName
	return WorkSubjects{
		Publish: []string{
			jetStreamInfo,
			streamInfoStart + scrapeRequests,
			streamInfoStart + pageOffers,
			streamInfoStart + pagerwicontract.PageRWIsStreamName,
			consumerInfoStart + scrapeRequests + "." + scrapeRequestDurable,
			consumerInfoStart + pageOffers + "." + pageOfferDurable,
			consumerListStart + pageOffers,
			consumerListStart + pagerwicontract.PageRWIsStreamName,
			nextMessageStart + scrapeRequests + "." + scrapeRequestDurable,
			nextMessageStart + pageOffers + "." + pageOfferDurable,
			ackStart + scrapeRequests + "." + scrapeRequestDurable + ".>",
			ackStart + pageOffers + "." + pageOfferDurable + ".>",
			pagescrapecontract.OfferedPageSubject,
			pagescrapecontract.ScrapeFailureSubject,
			pagescrapecontract.ScrapeOutcomeSubjectPrefix + ".>",
			pagescrapecontract.EveryIntakeReceiptSubject,
			pagerwicontract.PageRWISubject,
		},
		Subscribe: []string{pagescrapecontract.EveryIntakeReceiptSubject},
	}
}
