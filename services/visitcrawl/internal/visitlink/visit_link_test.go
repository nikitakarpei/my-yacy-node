package visitlink_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitlink"
)

const (
	secret      = "shared-secret"
	visitedPage = "https://example.org/a?q=1&r=2"
)

var expires = time.Unix(1700000000, 0)

func signatureOver(expires time.Time, visitedPage, secret string) string {
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(strconv.FormatInt(expires.Unix(), 10) + "\n" + visitedPage))
	return hex.EncodeToString(seal.Sum(nil))
}

func genuineLink() visitlink.VisitLink {
	return visitlink.VisitLink{
		VisitedPage: visitedPage,
		Expires:     expires,
		Signature:   signatureOver(expires, visitedPage, secret),
	}
}

func TestGenuineLinkIsGenuine(t *testing.T) {
	if !genuineLink().IsGenuine(secret) {
		t.Fatal("link signed with the shared secret is not genuine")
	}
}

func TestLinkSignedWithAnotherSecretIsNotGenuine(t *testing.T) {
	link := genuineLink()
	link.Signature = signatureOver(expires, visitedPage, "another-secret")
	if link.IsGenuine(secret) {
		t.Fatal("link signed with another secret is genuine")
	}
}

func TestLinkWithTamperedVisitedPageIsNotGenuine(t *testing.T) {
	link := genuineLink()
	link.VisitedPage = "https://evil.example/a?q=1&r=2"
	if link.IsGenuine(secret) {
		t.Fatal("link with a replaced visited page is genuine")
	}
}

func TestLinkWithTamperedExpiresIsNotGenuine(t *testing.T) {
	link := genuineLink()
	link.Expires = expires.Add(time.Hour)
	if link.IsGenuine(secret) {
		t.Fatal("link with a moved expiry is genuine")
	}
}

func TestLinkIsExpiredAfterItsExpires(t *testing.T) {
	link := genuineLink()
	if link.IsExpired(expires) {
		t.Fatal("link is expired at its own expiry")
	}
	if !link.IsExpired(expires.Add(time.Second)) {
		t.Fatal("link is not expired one second after its expiry")
	}
}
