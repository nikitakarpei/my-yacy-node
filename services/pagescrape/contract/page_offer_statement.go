package pagescrapecontract

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/intotostatements"
)

const (
	PageOfferPredicateType = "https://github.com/nikitakarpei/my-yacy-node/blob/main/" +
		"services/pagescrape/doc/page-offer-statement.md"
	offerSubjectName = "pageOffer"
)

var (
	ErrNotAPageOfferStatement = errors.New("not a page offer statement")
	pageOfferStatementKind    = intotostatements.Kind{
		SubjectName:   offerSubjectName,
		PredicateType: PageOfferPredicateType,
	}
)

type PageOfferStatement struct {
	OfferDigest  [sha256.Size]byte
	FetchURL     canonicalurl.CanonicalURL
	StartedAfter json.RawMessage
}

func PageOfferStatementOf(
	offerBody []byte,
	fetchURL canonicalurl.CanonicalURL,
	startedAfter json.RawMessage,
) PageOfferStatement {
	return PageOfferStatement{
		OfferDigest:  sha256.Sum256(offerBody),
		FetchURL:     fetchURL,
		StartedAfter: startedAfter,
	}
}

func (s PageOfferStatement) Covers(offerBody []byte) bool {
	return s.OfferDigest == sha256.Sum256(offerBody)
}

type pageOfferPredicateWire struct {
	FetchURL     canonicalurl.CanonicalURL `json:"fetchURL"`
	StartedAfter json.RawMessage           `json:"startedAfter"`
}

func MarshalPageOfferStatement(statement PageOfferStatement) []byte {
	predicate, _ := json.Marshal(pageOfferPredicateWire{ //nolint:errchkjson // a URL and raw JSON.
		FetchURL:     statement.FetchURL,
		StartedAfter: statement.StartedAfter,
	})

	return intotostatements.Marshal(intotostatements.Statement{
		Kind:          pageOfferStatementKind,
		SubjectDigest: statement.OfferDigest,
		Predicate:     predicate,
	})
}

func UnmarshalPageOfferStatement(data []byte) (PageOfferStatement, error) {
	statement, err := intotostatements.Unmarshal(data, pageOfferStatementKind)
	if err != nil {
		return PageOfferStatement{}, fmt.Errorf("%w: %w", ErrNotAPageOfferStatement, err)
	}
	var predicate pageOfferPredicateWire
	if err := json.Unmarshal(statement.Predicate, &predicate); err != nil {
		return PageOfferStatement{}, fmt.Errorf("%w: predicate: %w", ErrNotAPageOfferStatement, err)
	}

	return PageOfferStatement{
		OfferDigest:  statement.SubjectDigest,
		FetchURL:     predicate.FetchURL,
		StartedAfter: predicate.StartedAfter,
	}, nil
}
