package visitcrawl_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultlinks/visitcrawl"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestTheLinkSendsTheVisitThroughTheVisitPath(t *testing.T) {
	resultLinks := visitcrawl.New("secret", time.Hour, fixedClock{now: time.Unix(1_000, 0)})

	link, err := url.Parse(resultLinks.LinkFor("https://example.org/page?a=1&b=2"))
	if err != nil {
		t.Fatalf("parse the link: %v", err)
	}

	if link.Path != visitcrawl.Path {
		t.Errorf("path = %q, want %q", link.Path, visitcrawl.Path)
	}
	if link.Query().Get("url") != "https://example.org/page?a=1&b=2" {
		t.Errorf("url = %q, want the result page", link.Query().Get("url"))
	}
}

func TestTheLinkExpiresAfterItsLifetime(t *testing.T) {
	resultLinks := visitcrawl.New("secret", time.Hour, fixedClock{now: time.Unix(1_000, 0)})

	link, err := url.Parse(resultLinks.LinkFor("https://example.org/"))
	if err != nil {
		t.Fatalf("parse the link: %v", err)
	}

	if link.Query().Get("expires") != strconv.Itoa(1_000+3_600) {
		t.Errorf("expires = %q, want an hour after now", link.Query().Get("expires"))
	}
}

func TestTheSecretSignsTheExpiryAndThePage(t *testing.T) {
	resultLinks := visitcrawl.New("secret", time.Hour, fixedClock{now: time.Unix(1_000, 0)})

	link, err := url.Parse(resultLinks.LinkFor("https://example.org/"))
	if err != nil {
		t.Fatalf("parse the link: %v", err)
	}

	seal := hmac.New(sha256.New, []byte("secret"))
	seal.Write([]byte("4600\nhttps://example.org/"))
	if want := hex.EncodeToString(seal.Sum(nil)); link.Query().Get("signature") != want {
		t.Errorf("signature = %q, want %q", link.Query().Get("signature"), want)
	}
}
