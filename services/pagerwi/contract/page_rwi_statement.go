package pagerwicontract

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	PageRWIPredicateType = "https://github.com/nikitakarpei/my-yacy-node/blob/main/" +
		"services/pagerwi/doc/page-rwi-statement.md"
	InputVouchedBySameKey = "sameKey"
	inTotoStatementType   = "https://in-toto.io/Statement/v1"
	pageRWIRootSubject    = "pageRWIRoot"
)

var ErrNotAPageRWIStatement = errors.New("not a page RWI statement")

type PageRWIStatement struct {
	PageRWIRoot   yacymodel.PageRWIRoot
	PageURL       canonicalurl.CanonicalURL
	LandedURL     canonicalurl.CanonicalURL
	InputVouching string
	StartedAfter  json.RawMessage
}

func PageRWIStatementOf(
	root yacymodel.PageRWIRoot,
	pageURL canonicalurl.CanonicalURL,
	landedURL canonicalurl.CanonicalURL,
	startedAfter json.RawMessage,
) PageRWIStatement {
	return PageRWIStatement{
		PageRWIRoot:   root,
		PageURL:       pageURL,
		LandedURL:     landedURL,
		InputVouching: InputVouchedBySameKey,
		StartedAfter:  startedAfter,
	}
}

type pageRWIStatementWire struct {
	Type          string                 `json:"_type"`
	Subject       []statementSubjectWire `json:"subject"`
	PredicateType string                 `json:"predicateType"`
	Predicate     pageRWIPredicateWire   `json:"predicate"`
}

type statementSubjectWire struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

type pageRWIPredicateWire struct {
	PageURL       canonicalurl.CanonicalURL `json:"pageURL"`
	LandedURL     canonicalurl.CanonicalURL `json:"landedURL,omitzero"`
	InputVouching string                    `json:"inputVouching"`
	StartedAfter  json.RawMessage           `json:"startedAfter"`
}

func MarshalPageRWIStatement(statement PageRWIStatement) []byte {
	predicate := pageRWIPredicateWire{
		PageURL:       statement.PageURL,
		InputVouching: statement.InputVouching,
		StartedAfter:  statement.StartedAfter,
	}
	if statement.LandedURL != statement.PageURL {
		predicate.LandedURL = statement.LandedURL
	}
	data, _ := json.Marshal(pageRWIStatementWire{ //nolint:errchkjson // strings, URLs and raw JSON.
		Type: inTotoStatementType,
		Subject: []statementSubjectWire{{
			Name:   pageRWIRootSubject,
			Digest: map[string]string{"sha256": hex.EncodeToString(statement.PageRWIRoot.Bytes())},
		}},
		PredicateType: PageRWIPredicateType,
		Predicate:     predicate,
	})

	return data
}

func UnmarshalPageRWIStatement(data []byte) (PageRWIStatement, error) {
	var wire pageRWIStatementWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return PageRWIStatement{}, fmt.Errorf("unmarshal page RWI statement: %w", err)
	}
	root, err := pageRWIRootOf(wire)
	if err != nil {
		return PageRWIStatement{}, err
	}
	landedURL := wire.Predicate.LandedURL
	if landedURL == (canonicalurl.CanonicalURL{}) {
		landedURL = wire.Predicate.PageURL
	}

	return PageRWIStatement{
		PageRWIRoot:   root,
		PageURL:       wire.Predicate.PageURL,
		LandedURL:     landedURL,
		InputVouching: wire.Predicate.InputVouching,
		StartedAfter:  wire.Predicate.StartedAfter,
	}, nil
}

func pageRWIRootOf(wire pageRWIStatementWire) (yacymodel.PageRWIRoot, error) {
	if wire.Type != inTotoStatementType || wire.PredicateType != PageRWIPredicateType ||
		len(wire.Subject) != 1 || wire.Subject[0].Name != pageRWIRootSubject {
		return yacymodel.PageRWIRoot{}, ErrNotAPageRWIStatement
	}
	raw, err := hex.DecodeString(wire.Subject[0].Digest["sha256"])
	if err != nil {
		return yacymodel.PageRWIRoot{}, fmt.Errorf("%w: %w", ErrNotAPageRWIStatement, err)
	}
	root, err := yacymodel.ParsePageRWIRootBytes(raw)
	if err != nil {
		return yacymodel.PageRWIRoot{}, fmt.Errorf("%w: %w", ErrNotAPageRWIStatement, err)
	}

	return root, nil
}
