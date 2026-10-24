// Package visitcrawl links each web result through /visit on the origin of the search page,
// with the signature and the expiry that visitcrawl accepts, so visitcrawl learns which
// result a person opens. A link stays the same for the whole day it is issued, so the
// browser can show a result as visited.
package visitcrawl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	visitPath    = "/visit"
	linkLifetime = 48 * time.Hour
	day          = 24 * time.Hour
)

type Clock interface {
	Now() time.Time
}

type VisitLinks struct {
	secret string
	clock  Clock
}

func New(secret string, clock Clock) VisitLinks {
	return VisitLinks{secret: secret, clock: clock}
}

func (links VisitLinks) LinkFor(address string) string {
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		return address
	}
	expires := strconv.FormatInt(expiryOf(links.clock.Now()).Unix(), 10)
	return visitPath + "?" + url.Values{
		"url":       {address},
		"expires":   {expires},
		"signature": {signatureOf(expires, address, links.secret)},
	}.Encode()
}

func expiryOf(issuedAt time.Time) time.Time {
	return issuedAt.UTC().Truncate(day).Add(linkLifetime)
}

func signatureOf(expires string, address string, secret string) string {
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(expires + "\n" + address))
	return hex.EncodeToString(seal.Sum(nil))
}
