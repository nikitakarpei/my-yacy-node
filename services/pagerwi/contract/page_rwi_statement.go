package pagerwicontract

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/intotostatements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	PageRWIPredicateType = "https://github.com/nikitakarpei/my-yacy-node/blob/main/" +
		"services/pagerwi/doc/page-rwi-statement.md"
	pageRWIRootSubject = "pageRWIRoot"
)

var (
	ErrNotAPageRWIStatement = errors.New("not a page RWI statement")
	pageRWIStatementKind    = intotostatements.Kind{
		SubjectName:   pageRWIRootSubject,
		PredicateType: PageRWIPredicateType,
	}
)

type PageRWIStatement struct {
	PageRWIRoot  yacymodel.PageRWIRoot
	PageURL      canonicalurl.CanonicalURL
	LandedURL    canonicalurl.CanonicalURL
	StartedAfter json.RawMessage
}

func PageRWIStatementOf(
	root yacymodel.PageRWIRoot,
	pageURL canonicalurl.CanonicalURL,
	landedURL canonicalurl.CanonicalURL,
	startedAfter json.RawMessage,
) PageRWIStatement {
	return PageRWIStatement{
		PageRWIRoot:  root,
		PageURL:      pageURL,
		LandedURL:    landedURL,
		StartedAfter: startedAfter,
	}
}

type pageRWIPredicateWire struct {
	PageURL      canonicalurl.CanonicalURL `json:"pageURL"`
	LandedURL    canonicalurl.CanonicalURL `json:"landedURL,omitzero"`
	StartedAfter json.RawMessage           `json:"startedAfter"`
}

func MarshalPageRWIStatement(statement PageRWIStatement) []byte {
	predicate := pageRWIPredicateWire{
		PageURL:      statement.PageURL,
		StartedAfter: statement.StartedAfter,
	}
	if statement.LandedURL != statement.PageURL {
		predicate.LandedURL = statement.LandedURL
	}
	predicateEncoding, _ := json.Marshal(predicate) //nolint:errchkjson // URLs and raw JSON.

	return intotostatements.Marshal(intotostatements.Statement{
		Kind:          pageRWIStatementKind,
		SubjectDigest: statement.PageRWIRoot.Digest(),
		Predicate:     predicateEncoding,
	})
}

func UnmarshalPageRWIStatement(data []byte) (PageRWIStatement, error) {
	statement, err := intotostatements.Unmarshal(data, pageRWIStatementKind)
	if err != nil {
		return PageRWIStatement{}, fmt.Errorf("%w: %w", ErrNotAPageRWIStatement, err)
	}
	var predicate pageRWIPredicateWire
	if err := json.Unmarshal(statement.Predicate, &predicate); err != nil {
		return PageRWIStatement{}, fmt.Errorf("%w: predicate: %w", ErrNotAPageRWIStatement, err)
	}
	landedURL := predicate.LandedURL
	if landedURL == (canonicalurl.CanonicalURL{}) {
		landedURL = predicate.PageURL
	}

	return PageRWIStatement{
		PageRWIRoot:  yacymodel.PageRWIRootFromDigest(statement.SubjectDigest),
		PageURL:      predicate.PageURL,
		LandedURL:    landedURL,
		StartedAfter: predicate.StartedAfter,
	}, nil
}
