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

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/offeredpages/attested"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	pageBody          = "<html>page</html>"
	ownSignature      = "signed by this key"
	captureRound      = 33667957
	roundSignature    = "ace44ad88121ed26163c00acaa4c89522295c4fb066a8e9712138c63644e43cadac17e3f29aa8056f8f4ef5449d6e16e"
	attestationBundle = `{"bundle":"of the offer"}`
)

var clockTime = time.Date(2026, time.November, 10, 9, 0, 0, 0, time.UTC)

func TestAPageEveryHopOfWhichThisKeyCapturedIsOfferedWithItsAttestation(t *testing.T) {
	drandChain := quicknet(t)
	request, landed := redirectedFetch(t, drandChain)
	signer := &fakeSigner{}

	page, ready := offeredPagesOver(signer, drandChain, &reasons{}).
		OfferedPageFrom(context.Background(), request, landed)

	if !ready || string(page.Attestation) != attestationBundle {
		t.Fatalf("page = %+v, ready = %v, want it ready with its attestation", page, ready)
	}
	if want := drandChain.RoundTime(captureRound); !page.FetchedAt.Equal(want) {
		t.Errorf("fetched at = %v, want the time of the capture round %v", page.FetchedAt, want)
	}
	statement, err := pagescrapecontract.UnmarshalPageOfferStatement(signer.signedStatement)
	if err != nil {
		t.Fatal(err)
	}
	unattested := page
	unattested.Attestation = nil
	wantStatement := pagescrapecontract.PageOfferStatementOf(
		unattested,
		request.FetchURL,
		statement.StartedAfter,
	)
	if string(
		signer.signedStatement,
	) != string(
		pagescrapecontract.MarshalPageOfferStatement(wantStatement),
	) {
		t.Errorf(
			"signed statement = %s, want the statement of the offered page",
			signer.signedStatement,
		)
	}
	if !strings.Contains(string(statement.StartedAfter), roundSignature) {
		t.Errorf("started after = %s, want the round of the capture", statement.StartedAfter)
	}
}

func TestAPageWhoseCapturesDoNotHoldIsOfferedWithoutAttestation(t *testing.T) {
	drandChain := quicknet(t)
	for reason, breakFetch := range map[string]func(*redirectfollowingfetch.LandedFetch){
		attested.ReasonCaptureMissing: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[0].Capture = pagefetch.Capture{}
		},
		attested.ReasonCaptureOfOtherSigner: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[1].Capture.Signature = []byte("signed by another key")
		},
		attested.ReasonCaptureUnreadable: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[1].Capture.Statement = []byte("no statement")
		},
		attested.ReasonCaptureOfOtherURL: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[0].URL = canonicalurltest.CanonicalURLOf(t, "https://example.org/other")
		},
		attested.ReasonCaptureOfOtherBody: func(l *redirectfollowingfetch.LandedFetch) {
			l.Outcome.Page.Body = []byte("other body")
		},
		attested.ReasonCaptureWithoutTime: func(l *redirectfollowingfetch.LandedFetch) {
			l.Hops[1].Capture.Statement = captureStatement(
				t, "https://example.org/b", pageBody, "0000000000000000000000000000000000000000000000000000000000000000",
			)
		},
	} {
		request, landed := redirectedFetch(t, drandChain)
		breakFetch(&landed)
		observed := &reasons{}

		page, ready := offeredPagesOver(&fakeSigner{}, drandChain, observed).
			OfferedPageFrom(context.Background(), request, landed)

		if !ready || page.Attestation != nil || !page.FetchedAt.Equal(clockTime) {
			t.Errorf("%s: page = %+v, ready = %v, want it ready, unattested, at the clock time",
				reason, page, ready)
		}
		if len(observed.reported) != 1 || observed.reported[0] != reason {
			t.Errorf("%s: reported %v", reason, observed.reported)
		}
	}
}

func TestAPageWithoutAnyHopIsOfferedWithoutAttestation(t *testing.T) {
	drandChain := quicknet(t)
	request, landed := redirectedFetch(t, drandChain)
	landed.Hops = nil
	observed := &reasons{}

	page, _ := offeredPagesOver(&fakeSigner{}, drandChain, observed).
		OfferedPageFrom(context.Background(), request, landed)

	if page.Attestation != nil || len(observed.reported) != 1 ||
		observed.reported[0] != attested.ReasonCaptureMissing {
		t.Errorf("page = %+v, reported %v", page, observed.reported)
	}
}

func TestAPageTheKeyDoesNotSignIsNotReady(t *testing.T) {
	drandChain := quicknet(t)
	request, landed := redirectedFetch(t, drandChain)

	_, ready := offeredPagesOver(&fakeSigner{refuses: true}, drandChain, &reasons{}).
		OfferedPageFrom(context.Background(), request, landed)

	if ready {
		t.Error("a page whose statement the key did not sign is ready to offer")
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
				hopCaptured(t, pageURL, "moved", drandChain.Hash),
				hopCaptured(t, landedURL, pageBody, drandChain.Hash),
			},
		}
}

func hopCaptured(
	t *testing.T,
	hopURL canonicalurl.CanonicalURL,
	body string,
	chain string,
) redirectfollowingfetch.Hop {
	t.Helper()
	httpsURL := "https://" + hopURL.Hostname() + hopURL.WebAddress().RequestURI()

	return redirectfollowingfetch.Hop{
		URL: hopURL,
		Capture: pagefetch.Capture{
			Statement:   captureStatement(t, httpsURL, body, chain),
			Signature:   []byte(ownSignature),
			KeyEvidence: []byte(`{"profile":"android"}`),
		},
	}
}

func captureStatement(t *testing.T, requestURL, body, chain string) []byte {
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
				"chain": chain, "round": captureRound, "signature": roundSignature,
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

func offeredPagesOver(
	signer *fakeSigner,
	drandChain verifier.DrandChain,
	observer *reasons,
) attested.OfferedPages {
	return attested.New(signer, drandChain, fixedClock{}, observer)
}

type fakeSigner struct {
	refuses         bool
	signedStatement []byte
}

func (s *fakeSigner) HasSigned(_ context.Context, _, signature []byte) bool {
	return string(signature) == ownSignature
}

func (s *fakeSigner) BundleOf(_ context.Context, statement []byte) ([]byte, bool) {
	if s.refuses {
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
