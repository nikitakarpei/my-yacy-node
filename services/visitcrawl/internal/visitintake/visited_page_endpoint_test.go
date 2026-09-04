package visitintake_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type recordingPlacement struct {
	order    yacycrawlcontract.CrawlOrder
	attempts int
}

func (p *recordingPlacement) Start(order yacycrawlcontract.CrawlOrder) {
	p.order = order
	p.attempts++
}

const linkSecret = "shared-secret"

func mount(placement visitintake.CrawlOrderPlacement) *http.ServeMux {
	mux := http.NewServeMux()
	profile := yacycrawlcontract.CrawlProfile{
		Scope: yacycrawlcontract.ScopeDomain,
	}
	visitintake.MountVisitIntake(mux, placement, profile, linkSecret)
	return mux
}

func signedTarget(visitedPage string, expires time.Time, secret string) string {
	seconds := strconv.FormatInt(expires.Unix(), 10)
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(seconds + "\n" + visitedPage))
	return fmt.Sprintf("%s?url=%s&expires=%s&signature=%s",
		visitintake.PathVisit, url.QueryEscape(visitedPage), seconds,
		hex.EncodeToString(seal.Sum(nil)))
}

func currentTarget(visitedPage string) string {
	return signedTarget(visitedPage, time.Now().Add(time.Minute), linkSecret)
}

func get(mux *http.ServeMux, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestVisitRedirectsAndAttemptsPlacement(t *testing.T) {
	placement := &recordingPlacement{}
	rec := get(mount(placement), currentTarget("https://example.org/a"))

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://example.org/a" {
		t.Fatalf("location = %q", got)
	}
	if placement.attempts != 1 {
		t.Fatalf("attempts = %d, want 1", placement.attempts)
	}
	if len(placement.order.SeedURLs) != 1 ||
		placement.order.SeedURLs[0] != canonicalurltest.CanonicalURLOf(t, "https://example.org/a") {
		t.Fatalf("order seeds = %v", placement.order.SeedURLs)
	}
	if placement.order.OrderID == "" {
		t.Fatal("order id is empty")
	}
}

func TestVisitRejectsMissingURL(t *testing.T) {
	placement := &recordingPlacement{}
	rec := get(mount(placement), visitintake.PathVisit)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if placement.attempts != 0 {
		t.Fatal("placement should not be attempted")
	}
}

func TestVisitRejectsMalformedLinks(t *testing.T) {
	for name, target := range map[string]string{
		"non-http scheme":  currentTarget("ftp://example.org/a"),
		"missing host":     currentTarget("https://"),
		"missing expires":  visitintake.PathVisit + "?url=https%3A%2F%2Fexample.org%2Fa&signature=ab",
		"unparsed expires": visitintake.PathVisit + "?url=https%3A%2F%2Fexample.org%2Fa&expires=soon&signature=ab",
		"missing signature": visitintake.PathVisit +
			"?url=https%3A%2F%2Fexample.org%2Fa&expires=9999999999",
	} {
		placement := &recordingPlacement{}
		rec := get(mount(placement), target)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, rec.Code)
		}
		if placement.attempts != 0 {
			t.Errorf("%s: placement should not be attempted", name)
		}
	}
}

func TestVisitRejectsLinkSignedWithAnotherSecret(t *testing.T) {
	placement := &recordingPlacement{}
	target := signedTarget("https://example.org/a", time.Now().Add(time.Minute), "another-secret")
	rec := get(mount(placement), target)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if placement.attempts != 0 {
		t.Fatal("placement should not be attempted")
	}
}

func TestVisitRejectsReplacedVisitedPage(t *testing.T) {
	placement := &recordingPlacement{}
	target := currentTarget("https://example.org/a")
	tampered := strings.Replace(target, "example.org", "evil.example", 1)
	rec := get(mount(placement), tampered)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if placement.attempts != 0 {
		t.Fatal("placement should not be attempted")
	}
}

func TestVisitRejectsExpiredLink(t *testing.T) {
	placement := &recordingPlacement{}
	target := signedTarget("https://example.org/a", time.Now().Add(-time.Minute), linkSecret)
	rec := get(mount(placement), target)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if placement.attempts != 0 {
		t.Fatal("placement should not be attempted")
	}
}

func TestVisitRejectsNonGet(t *testing.T) {
	mux := mount(&recordingPlacement{})
	req := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, visitintake.PathVisit, nil,
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
