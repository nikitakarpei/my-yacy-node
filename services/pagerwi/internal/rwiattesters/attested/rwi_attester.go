package attested

import (
	"context"
	"encoding/json"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	ReasonOfferUnattested        = "the offer has no attestation"
	ReasonOfferUnreadable        = "the attestation of the offer is unreadable"
	ReasonOfferOfOtherSigner     = "another key signed the offer"
	ReasonOfferNotCovered        = "the offer differs from its statement"
	ReasonFetchURLNotPageURL     = "the fetch URL is not the page URL"
	ReasonPageURLNotHTTPS        = "the page URL is not https on the default port"
	ReasonLandedURLTooLong       = "the landed URL is too long"
	ReasonOfferRoundOfOtherChain = "the offer has no round of the pinned drand chain"
	ReasonFetchTimeNotRoundTime  = "the fetch time is not the time of the drand round"
	ReasonPageWithoutPostings    = "the page has no postings"
)

const (
	maxLandedURLLength = 2048
	httpsScheme        = "https"
)

type StatementSigner interface {
	IsOwnSignature(ctx context.Context, statement, signature []byte) bool
	Sign(
		ctx context.Context,
		statement []byte,
		stamping attestedstatements.TimeStamping,
	) (attestedstatements.Bundle, bool)
}

type RWIAttestationObserver interface {
	RWINotAttested(ctx context.Context, pageURL canonicalurl.CanonicalURL, reason string)
}

type RWIAttester struct {
	signer     StatementSigner
	drandChain verifier.DrandChain
	observer   RWIAttestationObserver
}

func New(
	signer StatementSigner,
	drandChain verifier.DrandChain,
	observer RWIAttestationObserver,
) RWIAttester {
	return RWIAttester{signer: signer, drandChain: drandChain, observer: observer}
}

func (a RWIAttester) Attest(
	ctx context.Context,
	offer pagescrapecontract.PageOfferMessage,
	page pagescrapecontract.OfferedPage,
	pageRWI pagerwicontract.PageRWI,
) (attestedstatements.Bundle, bool) {
	attestable, reason := a.checkAttestable(ctx, offer, page, pageRWI)
	if reason != "" {
		a.observer.RWINotAttested(ctx, page.PageURL, reason)

		return nil, true
	}

	return a.signer.Sign(ctx, pagerwicontract.MarshalPageRWIStatement(
		pagerwicontract.PageRWIStatementOf(
			attestable.root, pageRWI.PageURL, page.LandedURL, attestable.startedAfter,
		),
	), attestedstatements.TimeStamped)
}

type attestableRWI struct {
	root         yacymodel.PageRWIRoot
	startedAfter json.RawMessage
}

func (a RWIAttester) checkAttestable(
	ctx context.Context,
	offer pagescrapecontract.PageOfferMessage,
	page pagescrapecontract.OfferedPage,
	pageRWI pagerwicontract.PageRWI,
) (attestableRWI, string) {
	offerStatement, reason := a.checkOffer(ctx, offer)
	if reason != "" {
		return attestableRWI{}, reason
	}
	if reason := checkAddresses(page, offerStatement); reason != "" {
		return attestableRWI{}, reason
	}
	if reason := a.checkRound(page, offerStatement); reason != "" {
		return attestableRWI{}, reason
	}
	root, rooted := yacymodel.PageRWIRootFrom(pageRWI.Metadata).Get()
	if !rooted {
		return attestableRWI{}, ReasonPageWithoutPostings
	}

	return attestableRWI{root: root, startedAfter: offerStatement.StartedAfter}, ""
}

func (a RWIAttester) checkOffer(
	ctx context.Context,
	offer pagescrapecontract.PageOfferMessage,
) (pagescrapecontract.PageOfferStatement, string) {
	if len(offer.Attestation) == 0 {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferUnattested
	}
	bundle, err := verifier.ParseBundle(offer.Attestation)
	if err != nil || len(bundle.DSSEEnvelope.Signatures) != 1 {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferUnreadable
	}
	statement := bundle.DSSEEnvelope.Payload
	if !a.signer.IsOwnSignature(ctx, statement, bundle.DSSEEnvelope.Signatures[0].Sig) {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferOfOtherSigner
	}
	offerStatement, err := pagescrapecontract.UnmarshalPageOfferStatement(statement)
	if err != nil {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferUnreadable
	}
	if !offerStatement.Covers(offer.Body) {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferNotCovered
	}

	return offerStatement, ""
}

func checkAddresses(
	page pagescrapecontract.OfferedPage,
	offerStatement pagescrapecontract.PageOfferStatement,
) string {
	pageAddress := page.PageURL.WebAddress()
	switch {
	case offerStatement.FetchURL != page.PageURL:
		return ReasonFetchURLNotPageURL
	case pageAddress.Scheme != httpsScheme || pageAddress.Port() != "":
		return ReasonPageURLNotHTTPS
	case len(page.LandedURL.String()) > maxLandedURLLength:
		return ReasonLandedURLTooLong
	default:
		return ""
	}
}

func (a RWIAttester) checkRound(
	page pagescrapecontract.OfferedPage,
	offerStatement pagescrapecontract.PageOfferStatement,
) string {
	var startedAfter verifier.StartedAfter
	if err := json.Unmarshal(offerStatement.StartedAfter, &startedAfter); err != nil ||
		startedAfter.Drand == nil || startedAfter.Drand.Chain != a.drandChain.Hash {
		return ReasonOfferRoundOfOtherChain
	}
	if !page.FetchedAt.Equal(a.drandChain.RoundTime(startedAfter.Drand.Round)) {
		return ReasonFetchTimeNotRoundTime
	}

	return ""
}
