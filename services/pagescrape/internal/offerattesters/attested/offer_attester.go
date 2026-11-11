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
	ReasonResponseUnattested        = "a hop has no response attestation"
	ReasonResponseOfOtherSigner     = "another key signed a response attestation"
	ReasonResponseUnreadable        = "a response attestation is unreadable"
	ReasonResponseOfOtherURL        = "a response attestation names another URL"
	ReasonResponseOfOtherBody       = "the last response attestation names another body"
	ReasonResponseRoundOfOtherChain = "the last response attestation has no round of the pinned drand chain"
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
	) (attestedstatements.Bundle, bool)
}

type Clock interface {
	Now() time.Time
}

type OfferAttestationObserver interface {
	OfferNotAttested(ctx context.Context, landedURL canonicalurl.CanonicalURL, reason string)
}

type OfferAttester struct {
	signer     StatementSigner
	drandChain verifier.DrandChain
	clock      Clock
	observer   OfferAttestationObserver
}

func New(
	signer StatementSigner,
	drandChain verifier.DrandChain,
	clock Clock,
	observer OfferAttestationObserver,
) OfferAttester {
	return OfferAttester{signer: signer, drandChain: drandChain, clock: clock, observer: observer}
}

func (a OfferAttester) FetchTimeOf(
	ctx context.Context,
	landed redirectfollowingfetch.LandedFetch,
) time.Time {
	startedAfter, reason := a.checkHops(ctx, landed)
	if reason != "" {
		return a.clock.Now()
	}

	return a.drandChain.RoundTime(startedAfter.Drand.Round)
}

func (a OfferAttester) checkHops(
	ctx context.Context,
	landed redirectfollowingfetch.LandedFetch,
) (*verifier.StartedAfter, string) {
	var lastStatement *verifier.Statement
	for _, hop := range landed.Hops {
		statement, reason := a.checkHop(ctx, hop)
		if reason != "" {
			return nil, reason
		}
		lastStatement = statement
	}
	if lastStatement == nil {
		return nil, ReasonResponseUnattested
	}
	if !namesBody(lastStatement, landed.Outcome.Page.Body) {
		return nil, ReasonResponseOfOtherBody
	}

	return a.checkStart(lastStatement)
}

func (a OfferAttester) checkHop(
	ctx context.Context,
	hop redirectfollowingfetch.Hop,
) (*verifier.Statement, string) {
	responseAttestation, attested := hop.Attestation.Get()
	if !attested {
		return nil, ReasonResponseUnattested
	}
	if !a.signer.IsOwnSignature(ctx, responseAttestation.Statement, responseAttestation.Signature) {
		return nil, ReasonResponseOfOtherSigner
	}
	statement, err := verifier.ParseStatement(responseAttestation.Statement)
	if err != nil {
		return nil, ReasonResponseUnreadable
	}
	predicate, err := httpresponse.Parse(statement)
	if err != nil {
		return nil, ReasonResponseUnreadable
	}
	if predicate.Request.URL != httpsFormOf(hop.URL) {
		return nil, ReasonResponseOfOtherURL
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

func (a OfferAttester) checkStart(statement *verifier.Statement) (*verifier.StartedAfter, string) {
	startedAfter, err := statement.StartedAfter()
	if err != nil || startedAfter == nil || startedAfter.Drand.Chain != a.drandChain.Hash {
		return nil, ReasonResponseRoundOfOtherChain
	}

	return startedAfter, ""
}

func (a OfferAttester) Attest(
	ctx context.Context,
	offer pagescrapecontract.PageOfferMessage,
	landed redirectfollowingfetch.LandedFetch,
) (attestedstatements.Bundle, bool) {
	startedAfter, reason := a.checkHops(ctx, landed)
	if reason != "" {
		a.observer.OfferNotAttested(ctx, landed.URL, reason)

		return nil, true
	}

	return a.signer.Sign(ctx, pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(
			offer.Body, landed.Hops[0].URL, encodingOf(startedAfter),
		),
	), attestedstatements.NotTimeStamped)
}

func encodingOf(startedAfter *verifier.StartedAfter) json.RawMessage {
	encoding, _ := json.Marshal(startedAfter) //nolint:errchkjson // strings and a number.

	return encoding
}
