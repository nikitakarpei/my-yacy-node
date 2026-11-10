package pagescrapecontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	PageOfferPredicateType = "https://github.com/nikitakarpei/my-yacy-node/blob/main/" +
		"services/pagescrape/doc/page-offer-statement.md"
	InputVouchedBySameKey = "sameKey"
	inTotoStatementType   = "https://in-toto.io/Statement/v1"
	bodySubjectName       = "body"
)

var ErrNotAPageOfferStatement = errors.New("not a page offer statement")

type PageOfferStatement struct {
	BodyDigest      [sha256.Size]byte
	PageURL         canonicalurl.CanonicalURL
	FetchURL        canonicalurl.CanonicalURL
	LandedURL       canonicalurl.CanonicalURL
	ContentType     string
	RobotsTagValues []string
	PageModifiedAt  time.Time
	InputVouching   string
	StartedAfter    json.RawMessage
}

func PageOfferStatementOf(
	page OfferedPage,
	fetchURL canonicalurl.CanonicalURL,
	startedAfter json.RawMessage,
) PageOfferStatement {
	return PageOfferStatement{
		BodyDigest:      sha256.Sum256(page.Body),
		PageURL:         page.PageURL,
		FetchURL:        fetchURL,
		LandedURL:       page.LandedURL,
		ContentType:     page.ContentType,
		RobotsTagValues: page.RobotsTagValues,
		PageModifiedAt:  page.PageModifiedAt,
		InputVouching:   InputVouchedBySameKey,
		StartedAfter:    startedAfter,
	}
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
	PageURL         canonicalurl.CanonicalURL `json:"pageURL"`
	FetchURL        canonicalurl.CanonicalURL `json:"fetchURL"`
	LandedURL       canonicalurl.CanonicalURL `json:"landedURL"`
	ContentType     string                    `json:"contentType"`
	RobotsTagValues []string                  `json:"robotsTagValues,omitzero"`
	PageModifiedAt  time.Time                 `json:"pageModifiedAt,omitzero"`
	InputVouching   string                    `json:"inputVouching"`
	StartedAfter    json.RawMessage           `json:"startedAfter"`
}

func MarshalPageOfferStatement(statement PageOfferStatement) []byte {
	data, _ := json.Marshal(
		pageOfferStatementWire{ //nolint:errchkjson // strings, URLs and raw JSON.
			Type: inTotoStatementType,
			Subject: []statementSubjectWire{{
				Name:   bodySubjectName,
				Digest: map[string]string{"sha256": hex.EncodeToString(statement.BodyDigest[:])},
			}},
			PredicateType: PageOfferPredicateType,
			Predicate: pageOfferPredicateWire{
				PageURL:         statement.PageURL,
				FetchURL:        statement.FetchURL,
				LandedURL:       statement.LandedURL,
				ContentType:     statement.ContentType,
				RobotsTagValues: statement.RobotsTagValues,
				PageModifiedAt:  statement.PageModifiedAt,
				InputVouching:   statement.InputVouching,
				StartedAfter:    statement.StartedAfter,
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
	bodyDigest, err := bodyDigestOf(wire)
	if err != nil {
		return PageOfferStatement{}, err
	}

	return PageOfferStatement{
		BodyDigest:      bodyDigest,
		PageURL:         wire.Predicate.PageURL,
		FetchURL:        wire.Predicate.FetchURL,
		LandedURL:       wire.Predicate.LandedURL,
		ContentType:     wire.Predicate.ContentType,
		RobotsTagValues: wire.Predicate.RobotsTagValues,
		PageModifiedAt:  wire.Predicate.PageModifiedAt,
		InputVouching:   wire.Predicate.InputVouching,
		StartedAfter:    wire.Predicate.StartedAfter,
	}, nil
}

func bodyDigestOf(wire pageOfferStatementWire) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	if wire.Type != inTotoStatementType || wire.PredicateType != PageOfferPredicateType ||
		len(wire.Subject) != 1 || wire.Subject[0].Name != bodySubjectName {
		return digest, ErrNotAPageOfferStatement
	}
	raw, err := hex.DecodeString(wire.Subject[0].Digest["sha256"])
	if err != nil || len(raw) != sha256.Size {
		return digest, fmt.Errorf("%w: body digest", ErrNotAPageOfferStatement)
	}
	copy(digest[:], raw)

	return digest, nil
}
