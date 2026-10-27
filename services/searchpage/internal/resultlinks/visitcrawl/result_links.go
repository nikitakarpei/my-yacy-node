// Package visitcrawl links each result through visitcrawl: the signed link sends a person on to
// the result page and lets visitcrawl turn the visit into a crawl order. visitcrawl accepts the
// link until it expires, and only with the secret that signed it.
package visitcrawl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

const (
	Path                = "/visit"
	queryParamURL       = "url"
	queryParamExpires   = "expires"
	queryParamSignature = "signature"
)

type Clock interface {
	Now() time.Time
}

type ResultLinks struct {
	clock    Clock
	secret   string
	lifetime time.Duration
}

func New(secret string, lifetime time.Duration, clock Clock) ResultLinks {
	return ResultLinks{clock: clock, secret: secret, lifetime: lifetime}
}

func (l ResultLinks) LinkFor(page string) string {
	expires := strconv.FormatInt(l.clock.Now().Add(l.lifetime).Unix(), 10)
	return Path + "?" + url.Values{
		queryParamURL:       {page},
		queryParamExpires:   {expires},
		queryParamSignature: {l.signatureOf(expires, page)},
	}.Encode()
}

func (l ResultLinks) signatureOf(expires string, page string) string {
	seal := hmac.New(sha256.New, []byte(l.secret))
	seal.Write([]byte(expires + "\n" + page))
	return hex.EncodeToString(seal.Sum(nil))
}
