package attested_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageattestations/attested"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	pageAddress    = "https://example.org/a"
	ownSignature   = "signed by this key"
	quicknetRound  = 33667957
	roundSignature = "ace44ad88121ed26163c00acaa4c89522295c4fb066a8e9712138c63644e43cadac17e3f29aa8056f8f4ef5449d6e16e"
	pageRWIBundle  = `{"bundle":"of the rwi"}`
	otherSignature = "signed by another key"
)

func TestAnRWIOfAVouchedOfferGetsAStampedStatementOverItsPageRWIRoot(t *testing.T) {
	drandChain := quicknet(t)
	offer, page := attestedOffer(t, drandChain, offerChange{})
	signer := &fakeSigner{}
	pageRWI := pageRWIWithPostings(t, page)

	attestation, ready := attested.New(signer, drandChain, &reasons{}).
		AttestationOf(context.Background(), offer, page, pageRWI)

	if !ready || string(attestation) != pageRWIBundle {
		t.Fatalf("attestation = %s, ready = %v", attestation, ready)
	}
	statement, err := pagerwicontract.UnmarshalPageRWIStatement(signer.stampedStatement)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := yacymodel.PageRWIRootFrom(pageRWI.Metadata).Get()
	if statement.PageRWIRoot != root || statement.PageURL != page.PageURL ||
		statement.LandedURL != page.LandedURL ||
		!strings.Contains(string(statement.StartedAfter), roundSignature) {
		t.Errorf("statement = %+v, want the root, the URLs and the round of the offer", statement)
	}
}

func TestAnRWIOfAnOfferThatDoesNotHoldGoesWithoutAttestation(t *testing.T) {
	drandChain := quicknet(t)
	for reason, change := range map[string]offerChange{
		attested.ReasonOfferUnattested:         {noAttestation: true},
		attested.ReasonOfferUnreadable:         {attestation: `{"no":"bundle"}`},
		attested.ReasonOfferOfOtherSigner:      {signature: otherSignature},
		attested.ReasonOfferUnlikeItsStatement: {bodyAfterSigning: "other body"},
		attested.ReasonFetchedElsewhere:        {fetchURL: "https://archive.example/a"},
		attested.ReasonPageURLNotHTTPS:         {pageURL: "http://example.org/a"},
		attested.ReasonLandedURLTooLong:        {landedURL: pageAddress + "?" + strings.Repeat("x", 2048)},
		attested.ReasonRoundOfOtherChain:       {roundChain: strings.Repeat("0", 64)},
		attested.ReasonFetchTimeNotRoundTime:   {fetchedAt: time.Unix(1, 0)},
	} {
		offer, page := attestedOffer(t, drandChain, change)
		observed := &reasons{}

		attestation, ready := attested.New(&fakeSigner{}, drandChain, observed).
			AttestationOf(context.Background(), offer, page, pageRWIWithPostings(t, page))

		if !ready || attestation != nil {
			t.Errorf("%s: attestation = %s, ready = %v", reason, attestation, ready)
		}
		if len(observed.reported) != 1 || observed.reported[0] != reason {
			t.Errorf("%s: reported %v", reason, observed.reported)
		}
	}
}

func TestAnRWIWithoutPostingsGoesWithoutAttestation(t *testing.T) {
	drandChain := quicknet(t)
	offer, page := attestedOffer(t, drandChain, offerChange{})
	observed := &reasons{}

	attestation, _ := attested.New(&fakeSigner{}, drandChain, observed).
		AttestationOf(context.Background(), offer, page, pagerwicontract.PageRWI{PageURL: page.PageURL})

	if attestation != nil || len(observed.reported) != 1 ||
		observed.reported[0] != attested.ReasonPageWithoutPostings {
		t.Errorf("attestation = %s, reported %v", attestation, observed.reported)
	}
}

func TestAnRWIWhoseStatementTheKeyDoesNotSignIsNotReady(t *testing.T) {
	drandChain := quicknet(t)
	offer, page := attestedOffer(t, drandChain, offerChange{})

	_, ready := attested.New(&fakeSigner{refuses: true}, drandChain, &reasons{}).
		AttestationOf(context.Background(), offer, page, pageRWIWithPostings(t, page))

	if ready {
		t.Error("an RWI whose statement the key did not sign is ready to publish")
	}
}

type offerChange struct {
	noAttestation    bool
	attestation      string
	signature        string
	bodyAfterSigning string
	fetchURL         string
	pageURL          string
	landedURL        string
	roundChain       string
	fetchedAt        time.Time
}

func attestedOffer(
	t *testing.T,
	drandChain verifier.DrandChain,
	change offerChange,
) (pagescrapecontract.PageOfferMessage, pagescrapecontract.OfferedPage) {
	t.Helper()
	pageURL := canonicalurltest.CanonicalURLOf(t, orElse(change.pageURL, pageAddress))
	page := pagescrapecontract.OfferedPage{
		PageURL: pageURL,
		LandedURL: canonicalurltest.CanonicalURLOf(
			t,
			orElse(change.landedURL, "https://example.org/b"),
		),
		ContentType: "text/html",
		Body:        []byte("<html>page</html>"),
		FetchedAt:   drandChain.RoundTime(quicknetRound),
	}
	if !change.fetchedAt.IsZero() {
		page.FetchedAt = change.fetchedAt
	}
	fetchURL := pageURL
	if change.fetchURL != "" {
		fetchURL = canonicalurltest.CanonicalURLOf(t, change.fetchURL)
	}
	offer := pagescrapecontract.PageOfferMessageOf(page)
	statement := pagescrapecontract.MarshalPageOfferStatement(
		pagescrapecontract.PageOfferStatementOf(
			offer.Body, fetchURL, startedAfter(orElse(change.roundChain, drandChain.Hash)),
		),
	)
	if change.bodyAfterSigning != "" {
		page.Body = []byte(change.bodyAfterSigning)
		offer = pagescrapecontract.PageOfferMessageOf(page)
	}
	if change.noAttestation {
		return offer, page
	}

	return offer.AttestedBy([]byte(orElse(
		change.attestation,
		bundleOf(t, statement, orElse(change.signature, ownSignature)),
	))), page
}

func startedAfter(chain string) json.RawMessage {
	return json.RawMessage(`{"drand":{"chain":"` + chain + `","round":` +
		"33667957" + `,"signature":"` + roundSignature + `"}}`)
}

func bundleOf(t *testing.T, statement []byte, signature string) string {
	t.Helper()
	bundle, err := json.Marshal(map[string]any{
		"mediaType": verifier.BundleMediaType,
		"dsseEnvelope": map[string]any{
			"payloadType": verifier.InTotoPayload,
			"payload":     base64.StdEncoding.EncodeToString(statement),
			"signatures": []map[string]string{
				{"sig": base64.StdEncoding.EncodeToString([]byte(signature))},
			},
		},
		"verificationMaterial": map[string]any{"key": map[string]any{"profile": "android"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	return string(bundle)
}

func pageRWIWithPostings(
	t *testing.T,
	page pagescrapecontract.OfferedPage,
) pagerwicontract.PageRWI {
	t.Helper()
	urlHash := yacymodel.URLHashOf(page.PageURL.WebAddress())
	posting := yacymodel.RWIPosting{WordHash: yacymodel.WordHash("page"), URLHash: urlHash}
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{posting}, yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		t.Fatal(err)
	}

	return pagerwicontract.PageRWI{
		PageURL: page.PageURL,
		Metadata: yacymodel.URLMetadata{
			Hash:         urlHash,
			Address:      page.PageURL.String(),
			PostingsRoot: pagePostings.Root(),
		},
		Postings: pagePostings.Postings(),
	}
}

func orElse(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func quicknet(t *testing.T) verifier.DrandChain {
	t.Helper()
	chain, err := verifier.DrandQuicknet()
	if err != nil {
		t.Fatal(err)
	}

	return chain
}

type fakeSigner struct {
	refuses          bool
	stampedStatement []byte
}

func (s *fakeSigner) HasSigned(_ context.Context, _, signature []byte) bool {
	return string(signature) == ownSignature
}

func (s *fakeSigner) StampedBundleOf(_ context.Context, statement []byte) ([]byte, bool) {
	if s.refuses {
		return nil, false
	}
	s.stampedStatement = statement

	return []byte(pageRWIBundle), true
}

type reasons struct {
	reported []string
}

func (r *reasons) RWINotAttested(_ context.Context, _ canonicalurl.CanonicalURL, reason string) {
	r.reported = append(r.reported, reason)
}
