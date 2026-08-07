// Package visitlink owns the signed link a link issuer hands to a browser for
// one visited page, and the rules that say whether that link is genuine and
// still current. VisitLink is its only surface.
package visitlink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

type VisitLink struct {
	VisitedPage string
	Expires     time.Time
	Signature   string
}

func (l VisitLink) IsGenuine(secret string) bool {
	return hmac.Equal([]byte(l.Signature), []byte(signatureOf(l, secret)))
}

func signatureOf(link VisitLink, secret string) string {
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(strconv.FormatInt(link.Expires.Unix(), 10) + "\n" + link.VisitedPage))
	return hex.EncodeToString(seal.Sum(nil))
}

func (l VisitLink) IsExpired(now time.Time) bool {
	return now.After(l.Expires)
}
