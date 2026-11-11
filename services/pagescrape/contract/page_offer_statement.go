package pagescrapecontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	PageOfferPredicateType = "https://github.com/nikitakarpei/my-yacy-node/blob/main/" +
		"services/pagescrape/doc/page-offer-statement.md"
	inTotoStatementType  = "https://in-toto.io/Statement/v1"
	pageOfferSubjectName = "pageOffer"
)

var ErrNotAPageOfferStatement = errors.New("not a page offer statement")

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

type pageOfferStatementWire struct {
	Type          string                 `json:"_type"`
	Subject       []statementSubjectWire `json:"subject"`
	PredicateType string                 `json:"predicateType"`
	Predicate     pageOfferPredicateWire `json:"predicate"`
}

type statementSubjectWire struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

type pageOfferPredicateWire struct {
	FetchURL     canonicalurl.CanonicalURL `json:"fetchURL"`
	StartedAfter json.RawMessage           `json:"startedAfter"`
}

func MarshalPageOfferStatement(statement PageOfferStatement) []byte {
	data, _ := json.Marshal(
		pageOfferStatementWire{ //nolint:errchkjson // strings, a URL and raw JSON.
			Type: inTotoStatementType,
			Subject: []statementSubjectWire{{
				Name:   pageOfferSubjectName,
				Digest: map[string]string{"sha256": hex.EncodeToString(statement.OfferDigest[:])},
			}},
			PredicateType: PageOfferPredicateType,
			Predicate: pageOfferPredicateWire{
				FetchURL:     statement.FetchURL,
				StartedAfter: statement.StartedAfter,
			},
		},
	)

	return data
}

func UnmarshalPageOfferStatement(data []byte) (PageOfferStatement, error) {
	var wire pageOfferStatementWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return PageOfferStatement{}, fmt.Errorf("unmarshal page offer statement: %w", err)
	}
	if !wire.isPageOfferStatement() {
		return PageOfferStatement{}, ErrNotAPageOfferStatement
	}
	offerDigest, err := offerDigestFrom(wire.Subject[0])
	if err != nil {
		return PageOfferStatement{}, err
	}

	return PageOfferStatement{
		OfferDigest:  offerDigest,
		FetchURL:     wire.Predicate.FetchURL,
		StartedAfter: wire.Predicate.StartedAfter,
	}, nil
}

func (w pageOfferStatementWire) isPageOfferStatement() bool {
	return w.Type == inTotoStatementType && w.PredicateType == PageOfferPredicateType &&
		len(w.Subject) == 1 && w.Subject[0].Name == pageOfferSubjectName
}

func offerDigestFrom(subject statementSubjectWire) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	raw, err := hex.DecodeString(subject.Digest["sha256"])
	if err != nil || len(raw) != sha256.Size {
		return digest, fmt.Errorf("%w: offer digest", ErrNotAPageOfferStatement)
	}
	copy(digest[:], raw)

	return digest, nil
}
