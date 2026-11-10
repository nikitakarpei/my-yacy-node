package attested

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/predicates/httpresponse"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	ReasonAttestationMissing       = "a hop has no response attestation"
	ReasonAttestationOfOtherSigner = "another key signed a response attestation"
	ReasonAttestationUnreadable    = "a response attestation is unreadable"
	ReasonAttestationOfOtherURL    = "a response attestation names another URL"
	ReasonAttestationOfOtherBody   = "the last response attestation names another body"
	ReasonAttestationWithoutTime   = "the last response attestation has no round of the pinned drand chain"
)

const (
	httpsScheme     = "https"
	bodySubjectName = "body"
)

type StatementSigner interface {
	IsOwnSignature(ctx context.Context, statement, signature []byte) bool
	Sign(
		ctx context.Context,
		statement []byte,
		stamping attestedstatements.TimeStamping,
	) ([]byte, bool)
}

type Clock interface {
	Now() time.Time
}

type AttestationObserver interface {
	OfferNotAttested(ctx context.Context, pageURL canonicalurl.CanonicalURL, reason string)
}

type OfferIssuer struct {
	signer     StatementSigner
	drandChain verifier.DrandChain
	clock      Clock
	observer   AttestationObserver
}

func New(
	signer StatementSigner,
	drandChain verifier.DrandChain,
	clock Clock,
	observer AttestationObserver,
) OfferIssuer {
	return OfferIssuer{signer: signer, drandChain: drandChain, clock: clock, observer: observer}
}

type captureStart struct {
	startedAfter json.RawMessage
	fetchedAt    time.Time
}

func (p OfferIssuer) Issue(
	ctx context.Context,
	request pagescrapecontract.ScrapeRequest,
	landed redirectfollowingfetch.LandedFetch,
) (pagescrapecontract.PageOfferMessage, bool) {
	start, reason := p.checkHops(ctx, landed)
	if reason != "" {
		p.observer.OfferNotAttested(ctx, request.PageURL, reason)

		return offerMessageFetchedAt(request, landed, p.clock.Now()), true
	}
	offer := offerMessageFetchedAt(request, landed, start.fetchedAt)
	attestation, signed := p.signer.Sign(ctx, pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(offer.Body, request.FetchURL, start.startedAfter),
	), attestedstatements.NotTimeStamped)
	if !signed {
		return offer, false
	}

	return offer.AttestedBy(attestation), true
}

func (p OfferIssuer) checkHops(
	ctx context.Context,
	landed redirectfollowingfetch.LandedFetch,
) (captureStart, string) {
	var lastStatement *verifier.Statement
	for _, hop := range landed.Hops {
		statement, reason := p.checkHop(ctx, hop)
		if reason != "" {
			return captureStart{}, reason
		}
		lastStatement = statement
	}
	if lastStatement == nil {
		return captureStart{}, ReasonAttestationMissing
	}
	if !namesBody(lastStatement, landed.Outcome.Page.Body) {
		return captureStart{}, ReasonAttestationOfOtherBody
	}

	return p.checkStart(lastStatement)
}

func (p OfferIssuer) checkHop(
	ctx context.Context,
	hop redirectfollowingfetch.Hop,
) (*verifier.Statement, string) {
	attestation, attested := hop.Attestation.Get()
	if !attested {
		return nil, ReasonAttestationMissing
	}
	if !p.signer.IsOwnSignature(ctx, attestation.Statement, attestation.Signature) {
		return nil, ReasonAttestationOfOtherSigner
	}
	statement, err := verifier.ParseStatement(attestation.Statement)
	if err != nil {
		return nil, ReasonAttestationUnreadable
	}
	predicate, err := httpresponse.Parse(statement)
	if err != nil {
		return nil, ReasonAttestationUnreadable
	}
	if predicate.Request.URL != httpsFormOf(hop.URL) {
		return nil, ReasonAttestationOfOtherURL
	}

	return statement, ""
}

func httpsFormOf(address canonicalurl.CanonicalURL) string {
	httpsAddress := *address.WebAddress()
	httpsAddress.Scheme = httpsScheme
	httpsAddress.Host = httpsAddress.Hostname()
	if strings.Contains(httpsAddress.Host, ":") {
		httpsAddress.Host = "[" + httpsAddress.Host + "]"
	}

	return httpsAddress.String()
}

func namesBody(statement *verifier.Statement, body []byte) bool {
	digest, named := statement.SubjectDigest(bodySubjectName)
	bodyDigest := sha256.Sum256(body)

	return named && digest == hex.EncodeToString(bodyDigest[:])
}

func (p OfferIssuer) checkStart(statement *verifier.Statement) (captureStart, string) {
	startedAfter, err := statement.StartedAfter()
	if err != nil || startedAfter == nil || startedAfter.Drand.Chain != p.drandChain.Hash {
		return captureStart{}, ReasonAttestationWithoutTime
	}
	encoding, _ := json.Marshal(startedAfter) //nolint:errchkjson // strings and a number.

	return captureStart{
		startedAfter: encoding,
		fetchedAt:    p.drandChain.RoundTime(startedAfter.Drand.Round),
	}, ""
}

func offerMessageFetchedAt(
	request pagescrapecontract.ScrapeRequest,
	landed redirectfollowingfetch.LandedFetch,
	fetchedAt time.Time,
) pagescrapecontract.PageOfferMessage {
	return pagescrapecontract.PageOfferMessageOf(pagescrapecontract.OfferedPageFrom(
		request, landed.Outcome.Page, landed.Outcome.Version, landed.URL, fetchedAt,
	))
}
