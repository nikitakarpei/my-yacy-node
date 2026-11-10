package attested

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	ReasonOfferUnattested         = "the offer has no attestation"
	ReasonOfferUnreadable         = "the attestation of the offer is unreadable"
	ReasonOfferOfOtherSigner      = "another key signed the offer"
	ReasonOfferUnlikeItsStatement = "the offer differs from its statement"
	ReasonFetchedElsewhere        = "the fetch URL is not the page URL"
	ReasonPageURLNotHTTPS         = "the page URL is not https on the default port"
	ReasonLandedURLTooLong        = "the landed URL is too long"
	ReasonRoundInvalid            = "the drand round of the offer does not verify"
	ReasonFetchTimeNotRoundTime   = "the fetch time is not the time of the drand round"
	ReasonPageWithoutPostings     = "the page has no postings"
)

const (
	maxLandedURLLength = 2048
	httpsScheme        = "https"
)

type StatementSigner interface {
	HasSigned(ctx context.Context, statement, signature []byte) bool
	StampedBundleOf(ctx context.Context, statement []byte) ([]byte, bool)
}

type AttestationObserver interface {
	RWINotAttested(ctx context.Context, pageURL canonicalurl.CanonicalURL, reason string)
}

type PageAttestations struct {
	signer     StatementSigner
	drandChain verifier.DrandChain
	observer   AttestationObserver
}

func New(
	signer StatementSigner,
	drandChain verifier.DrandChain,
	observer AttestationObserver,
) PageAttestations {
	return PageAttestations{signer: signer, drandChain: drandChain, observer: observer}
}

func (a PageAttestations) AttestationOf(
	ctx context.Context,
	page pagescrapecontract.OfferedPage,
	pageRWI pagerwicontract.PageRWI,
) (json.RawMessage, bool) {
	attestable, reason := a.attestableOf(ctx, page, pageRWI)
	if reason != "" {
		a.observer.RWINotAttested(ctx, page.PageURL, reason)

		return nil, true
	}

	return a.signer.StampedBundleOf(ctx, pagerwicontract.MarshalPageRWIStatement(
		pagerwicontract.PageRWIStatementOf(
			attestable.root, pageRWI.PageURL, page.LandedURL, attestable.startedAfter,
		),
	))
}

type attestableRWI struct {
	root         yacymodel.PageRWIRoot
	startedAfter json.RawMessage
}

func (a PageAttestations) attestableOf(
	ctx context.Context,
	page pagescrapecontract.OfferedPage,
	pageRWI pagerwicontract.PageRWI,
) (attestableRWI, string) {
	offerStatement, reason := a.vouchedOfferStatementOf(ctx, page)
	if reason == "" {
		reason = addressReasonOf(page, offerStatement)
	}
	if reason == "" {
		reason = a.roundReasonOf(page, offerStatement)
	}
	root, rooted := yacymodel.PageRWIRootFrom(pageRWI.Metadata).Get()
	if reason == "" && !rooted {
		reason = ReasonPageWithoutPostings
	}

	return attestableRWI{root: root, startedAfter: offerStatement.StartedAfter}, reason
}

func (a PageAttestations) vouchedOfferStatementOf(
	ctx context.Context,
	page pagescrapecontract.OfferedPage,
) (pagescrapecontract.PageOfferStatement, string) {
	if len(page.Attestation) == 0 {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferUnattested
	}
	bundle, err := verifier.ParseBundle(page.Attestation)
	if err != nil || len(bundle.DSSEEnvelope.Signatures) != 1 {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferUnreadable
	}
	statement := bundle.DSSEEnvelope.Payload
	if !a.signer.HasSigned(ctx, statement, bundle.DSSEEnvelope.Signatures[0].Sig) {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferOfOtherSigner
	}
	offerStatement, err := pagescrapecontract.UnmarshalPageOfferStatement(statement)
	if err != nil {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferUnreadable
	}
	if !bytes.Equal(statement, pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(
			page, offerStatement.FetchURL, offerStatement.StartedAfter,
		),
	)) {
		return pagescrapecontract.PageOfferStatement{}, ReasonOfferUnlikeItsStatement
	}

	return offerStatement, ""
}

func addressReasonOf(
	page pagescrapecontract.OfferedPage,
	offerStatement pagescrapecontract.PageOfferStatement,
) string {
	pageAddress := page.PageURL.WebAddress()
	switch {
	case offerStatement.FetchURL != page.PageURL:
		return ReasonFetchedElsewhere
	case pageAddress.Scheme != httpsScheme || pageAddress.Port() != "":
		return ReasonPageURLNotHTTPS
	case len(page.LandedURL.String()) > maxLandedURLLength:
		return ReasonLandedURLTooLong
	default:
		return ""
	}
}

func (a PageAttestations) roundReasonOf(
	page pagescrapecontract.OfferedPage,
	offerStatement pagescrapecontract.PageOfferStatement,
) string {
	var startedAfter verifier.StartedAfter
	if err := json.Unmarshal(offerStatement.StartedAfter, &startedAfter); err != nil ||
		startedAfter.Drand == nil || startedAfter.Drand.Chain != a.drandChain.Hash {
		return ReasonRoundInvalid
	}
	signature, err := hex.DecodeString(startedAfter.Drand.Signature)
	if err != nil || a.drandChain.Verify(startedAfter.Drand.Round, signature) != nil {
		return ReasonRoundInvalid
	}
	if !page.FetchedAt.Equal(a.drandChain.RoundTime(startedAfter.Drand.Round)) {
		return ReasonFetchTimeNotRoundTime
	}

	return ""
}
