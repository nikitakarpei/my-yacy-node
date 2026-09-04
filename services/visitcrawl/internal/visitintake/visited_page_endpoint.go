package visitintake

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitlink"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	queryParamURL       = "url"
	queryParamExpires   = "expires"
	queryParamSignature = "signature"
)

var (
	errForgedSignature = errors.New(queryParamSignature + ": does not match the link")
	errExpiredLink     = errors.New(queryParamExpires + ": link has expired")
)

type visitedPageEndpoint struct {
	placement  CrawlOrderPlacement
	profile    yacycrawlcontract.CrawlProfile
	linkSecret string
}

func (e visitedPageEndpoint) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	link, err := visitLinkFrom(req.URL.Query())
	if err != nil {
		e.rejectVisit(req.Context(), w, err)
		return
	}
	if !link.IsGenuine(e.linkSecret) {
		e.rejectVisit(req.Context(), w, errForgedSignature)
		return
	}
	if link.IsExpired(time.Now()) {
		e.rejectVisit(req.Context(), w, errExpiredLink)
		return
	}
	seedURL, err := seedURLFrom(link.VisitedPage)
	if err != nil {
		e.rejectVisit(req.Context(), w, err)
		return
	}

	e.placement.Start(crawlOrderFor(seedURL, e.profile))

	http.Redirect(w, req, link.VisitedPage, http.StatusFound)
}

func visitLinkFrom(query url.Values) (visitlink.VisitLink, error) {
	visitedPage, err := visitedPageFrom(query.Get(queryParamURL))
	if err != nil {
		return visitlink.VisitLink{}, err
	}
	expires, err := expiresFrom(query.Get(queryParamExpires))
	if err != nil {
		return visitlink.VisitLink{}, err
	}
	signature, err := signatureFrom(query.Get(queryParamSignature))
	if err != nil {
		return visitlink.VisitLink{}, err
	}
	return visitlink.VisitLink{
		VisitedPage: visitedPage,
		Expires:     expires,
		Signature:   signature,
	}, nil
}

func visitedPageFrom(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%s: must be set", queryParamURL)
	}
	return raw, nil
}

func seedURLFrom(visitedPage string) (canonicalurl.CanonicalURL, error) {
	canonicalURL, err := canonicalurl.CanonicalURLOf(visitedPage)
	if err != nil {
		return canonicalurl.CanonicalURL{}, fmt.Errorf("%s: %w", queryParamURL, err)
	}
	return canonicalURL, nil
}

func expiresFrom(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("%s: must be set", queryParamExpires)
	}
	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: must be unix seconds", queryParamExpires)
	}
	return time.Unix(seconds, 0), nil
}

func signatureFrom(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%s: must be set", queryParamSignature)
	}
	return raw, nil
}

func (e visitedPageEndpoint) rejectVisit(
	ctx context.Context,
	w http.ResponseWriter,
	err error,
) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}
