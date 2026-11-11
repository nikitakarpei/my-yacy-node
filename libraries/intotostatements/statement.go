package intotostatements

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	statementType   = "https://in-toto.io/Statement/v1"
	digestAlgorithm = "sha256"
)

var ErrOtherStatement = errors.New("not the expected in-toto statement")

type Kind struct {
	SubjectName   string
	PredicateType string
}

type Statement struct {
	Kind          Kind
	SubjectDigest [sha256.Size]byte
	Predicate     json.RawMessage
}

type statementWire struct {
	Type          string          `json:"_type"`
	Subject       []subjectWire   `json:"subject"`
	PredicateType string          `json:"predicateType"`
	Predicate     json.RawMessage `json:"predicate"`
}

type subjectWire struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

func Marshal(statement Statement) []byte {
	data, _ := json.Marshal(
		statementWire{ //nolint:errchkjson // strings and raw JSON from json.Marshal.
			Type: statementType,
			Subject: []subjectWire{
				{
					Name: statement.Kind.SubjectName,
					Digest: map[string]string{
						digestAlgorithm: hex.EncodeToString(statement.SubjectDigest[:]),
					},
				},
			},
			PredicateType: statement.Kind.PredicateType,
			Predicate:     statement.Predicate,
		},
	)

	return data
}

func Unmarshal(data []byte, kind Kind) (Statement, error) {
	var wire statementWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return Statement{}, fmt.Errorf("%w: %w", ErrOtherStatement, err)
	}
	if !wire.isOfKind(kind) {
		return Statement{}, ErrOtherStatement
	}
	digest, err := digestFrom(wire.Subject[0])
	if err != nil {
		return Statement{}, err
	}

	return Statement{Kind: kind, SubjectDigest: digest, Predicate: wire.Predicate}, nil
}

func (w statementWire) isOfKind(kind Kind) bool {
	return w.Type == statementType && w.PredicateType == kind.PredicateType &&
		len(w.Subject) == 1 && w.Subject[0].Name == kind.SubjectName
}

func digestFrom(subject subjectWire) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	raw, err := hex.DecodeString(subject.Digest[digestAlgorithm])
	if err != nil || len(raw) != sha256.Size {
		return digest, fmt.Errorf("%w: the subject has no SHA-256 digest", ErrOtherStatement)
	}
	copy(digest[:], raw)

	return digest, nil
}
