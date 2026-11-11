package attested_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/predicates/httpresponse"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/optional"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/offerattesters/attested"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	pageBody          = "<html>page</html>"
	ownSignature      = "signed by this key"
	quicknetRound     = 33667957
	roundSignature    = "ace44ad88121ed26163c00acaa4c89522295c4fb066a8e9712138c63644e43cadac17e3f29aa8056f8f4ef5449d6e16e"
	attestationBundle = `{"bundle":"of the offer"}`
)

var (
	clockTime = time.Date(2026, time.November, 10, 9, 0, 0, 0, time.UTC)
	offerBody = pagescrapecontract.PageOfferMessage{
		Body: []byte(`{"PageURL":"http://example.org/a"}`),
	}
)

func TestAPageEveryHopOfWhichThisKeyAttestedIsFetchedAtTheRoundTime(t *testing.T) {
	drandChain := quicknet(t)
	_, landed := redirectedFetch(t, drandChain)

	fetchedAt := offerAttesterOver(&fakeSigner{}, drandChain, &reasons{}).
		FetchTimeOf(context.Background(), landed)

	if want := drandChain.RoundTime(quicknetRound); !fetchedAt.Equal(want) {
		t.Errorf("fetched at = %v, want the time of the round %v", fetchedAt, want)
	}
}

func TestAnOfferEveryHopOfWhichThisKeyAttestedGetsItsBundle(t *testing.T) {
	drandChain := quicknet(t)
	request, landed := redirectedFetch(t, drandChain)
	signer := &fakeSigner{}

	bundle, decided := offerAttesterOver(signer, drandChain, &reasons{}).
		Attest(context.Background(), offerBody, landed)

	if !decided || string(bundle) != attestationBundle {
		t.Fatalf("bundle = %s, decided = %v, want the bundle of the offer", bundle, decided)
	}
	statement, err := pagescrapecontract.UnmarshalPageOfferStatement(signer.signedStatement)
	if err != nil {
		t.Fatal(err)
	}
	if !statement.Covers(offerBody.Body) || statement.FetchURL != request.FetchURL {
		t.Errorf(
			"signed statement = %s, want the statement of the offer body",
			signer.signedStatement,
		)
	}
	if !strings.Contains(string(statement.StartedAfter), roundSignature) {
		t.Errorf("started after = %s, want the round of the response", statement.StartedAfter)
	}
}

func TestAPageWhoseResponseAttestationsDoNotHoldIsFetchedAtTheClockTimeAndUnattested(t *testing.T) {
	drandChain := quicknet(t)
	for reason, breakFetch := range map[string]func(*redirectfollowingfetch.LandedFetch){
		attested.ReasonResponseUnattested: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[0].Attestation = optional.None[pagefetch.ResponseAttestation]()
		},
		attested.ReasonResponseOfOtherSigner: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[1].Attestation = withSignature(t, l.Hops[1], []byte("signed by another key"))
		},
		attested.ReasonResponseUnreadable: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[1].Attestation = withStatement(t, l.Hops[1], []byte("no statement"))
		},
		attested.ReasonResponseOfOtherURL: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[0].URL = canonicalurltest.CanonicalURLOf(t, "https://example.org/other")
		},
		attested.ReasonResponseOfOtherBody: func(l *redirectfollowingfetch.LandedFetch) {
			l.Outcome.Page.Body = []byte("other body")
		},
		attested.ReasonResponseRoundOfOtherChain: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[1].Attestation = withStatement(t, l.Hops[1], responseStatement(
				t, "https://example.org/b", pageBody, "0000000000000000000000000000000000000000000000000000000000000000",
			))
		},
	} {
		_, landed := redirectedFetch(t, drandChain)
		breakFetch(&landed)
		observed := &reasons{}
		offerAttester := offerAttesterOver(&fakeSigner{}, drandChain, observed)

		fetchedAt := offerAttester.FetchTimeOf(context.Background(), landed)
		bundle, decided := offerAttester.Attest(context.Background(), offerBody, landed)

		if !fetchedAt.Equal(clockTime) || !decided || len(bundle) != 0 {
			t.Errorf(
				"%s: fetched at %v, bundle = %s, decided = %v, want the clock time and no bundle",
				reason,
				fetchedAt,
				bundle,
				decided,
			)
		}
		if len(observed.reported) != 1 || observed.reported[0] != reason {
			t.Errorf("%s: reported %v", reason, observed.reported)
		}
	}
}

func TestAPageWithoutAnyHopIsUnattested(t *testing.T) {
	drandChain := quicknet(t)
	_, landed := redirectedFetch(t, drandChain)
	landed.Hops = nil
	observed := &reasons{}

	bundle, _ := offerAttesterOver(&fakeSigner{}, drandChain, observed).
		Attest(context.Background(), offerBody, landed)

	if len(bundle) != 0 || len(observed.reported) != 1 ||
		observed.reported[0] != attested.ReasonResponseUnattested {
		t.Errorf("bundle = %s, reported %v", bundle, observed.reported)
	}
}

func TestAnOfferTheKeyDoesNotSignIsNotDecided(t *testing.T) {
	drandChain := quicknet(t)
	_, landed := redirectedFetch(t, drandChain)

	_, decided := offerAttesterOver(&fakeSigner{refuses: true}, drandChain, &reasons{}).
		Attest(context.Background(), offerBody, landed)

	if decided {
		t.Error("an offer whose statement the key did not sign is decided")
	}
}

func redirectedFetch(
	t *testing.T,
	drandChain verifier.DrandChain,
) (pagescrapecontract.ScrapeRequest, redirectfollowingfetch.LandedFetch) {
	t.Helper()
	pageURL := canonicalurltest.CanonicalURLOf(t, "http://example.org/a")
	landedURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/b")

	return pagescrapecontract.ScrapeRequest{PageURL: pageURL, FetchURL: pageURL},
		redirectfollowingfetch.LandedFetch{
			URL: landedURL,
			Outcome: pagefetch.FetchOutcome{
				Status: pagefetch.FetchSucceeded,
				Page:   pagefetch.FetchedPage{ContentType: "text/html", Body: []byte(pageBody)},
			},
			Hops: []redirectfollowingfetch.Hop{
				hopAttested(t, pageURL, "moved", drandChain.Hash),
				hopAttested(t, landedURL, pageBody, drandChain.Hash),
			},
		}
}

func hopAttested(
	t *testing.T,
	hopURL canonicalurl.CanonicalURL,
	body string,
	chain string,
) redirectfollowingfetch.Hop {
	t.Helper()
	httpsURL := "https://" + hopURL.Hostname() + hopURL.WebAddress().RequestURI()

	return redirectfollowingfetch.Hop{
		URL: hopURL,
		Attestation: optional.Some(pagefetch.ResponseAttestation{
			Statement:   responseStatement(t, httpsURL, body, chain),
			Signature:   []byte(ownSignature),
			KeyEvidence: []byte(`{"profile":"android"}`),
		}),
	}
}

func withStatement(
	t *testing.T,
	hop redirectfollowingfetch.Hop,
	statement []byte,
) optional.Value[pagefetch.ResponseAttestation] {
	t.Helper()
	attestation, _ := hop.Attestation.Get()
	attestation.Statement = statement

	return optional.Some(attestation)
}

func withSignature(
	t *testing.T,
	hop redirectfollowingfetch.Hop,
	signature []byte,
) optional.Value[pagefetch.ResponseAttestation] {
	t.Helper()
	attestation, _ := hop.Attestation.Get()
	attestation.Signature = signature

	return optional.Some(attestation)
}

func responseStatement(t *testing.T, requestURL, body, chain string) []byte {
	t.Helper()
	bodyDigest := sha256.Sum256([]byte(body))
	statement, err := json.Marshal(map[string]any{
		"_type": verifier.InTotoStatement1,
		"subject": []map[string]any{{
			"name":   "body",
			"digest": map[string]string{"sha256": hex.EncodeToString(bodyDigest[:])},
		}},
		"predicateType": httpresponse.PredicateType,
		"predicate": map[string]any{
			"request":  map[string]any{"method": "GET", "url": requestURL, "cookies": "none"},
			"response": map[string]any{"statusLine": "HTTP/1.1 200 OK", "status": 200},
			"startedAfter": map[string]any{"drand": map[string]any{
				"chain": chain, "round": quicknetRound, "signature": roundSignature,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	return statement
}

func quicknet(t *testing.T) verifier.DrandChain {
	t.Helper()
	chain, err := verifier.DrandQuicknet()
	if err != nil {
		t.Fatal(err)
	}

	return chain
}

func offerAttesterOver(
	signer *fakeSigner,
	drandChain verifier.DrandChain,
	observer *reasons,
) attested.OfferAttester {
	return attested.New(signer, drandChain, fixedClock{}, observer)
}

type fakeSigner struct {
	refuses         bool
	signedStatement []byte
}

func (s *fakeSigner) IsOwnSignature(_ context.Context, _, signature []byte) bool {
	return string(signature) == ownSignature
}

func (s *fakeSigner) Sign(
	_ context.Context,
	statement []byte,
	stamping attestedstatements.TimeStamping,
) (attestedstatements.Bundle, bool) {
	if s.refuses || stamping != attestedstatements.NotTimeStamped {
		return nil, false
	}
	s.signedStatement = statement

	return []byte(attestationBundle), true
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return clockTime
}

type reasons struct {
	reported []string
}

func (r *reasons) OfferNotAttested(_ context.Context, _ canonicalurl.CanonicalURL, reason string) {
	r.reported = append(r.reported, reason)
}
