package intotostatements_test

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/intotostatements"
)

var kind = intotostatements.Kind{SubjectName: "page", PredicateType: "https://example.org/page"}

func TestAStatementRoundTrips(t *testing.T) {
	written := intotostatements.Statement{
		Kind:          kind,
		SubjectDigest: sha256.Sum256([]byte("page")),
		Predicate:     json.RawMessage(`{"url":"https://example.org/"}`),
	}

	read, err := intotostatements.Unmarshal(intotostatements.Marshal(written), kind)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(read, written) {
		t.Errorf("read = %+v, want %+v", read, written)
	}
}

func TestAStatementIsWrittenInTheInTotoFormat(t *testing.T) {
	data := intotostatements.Marshal(intotostatements.Statement{
		Kind:          kind,
		SubjectDigest: [sha256.Size]byte{1},
		Predicate:     json.RawMessage(`{}`),
	})

	want := `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"page","digest":{"sha256":"01` +
		strings.Repeat(
			"00",
			sha256.Size-1,
		) + `"}}],"predicateType":"https://example.org/page","predicate":{}}`
	if string(data) != want {
		t.Errorf("statement = %s, want %s", data, want)
	}
}

func TestUnmarshalRefusesAnotherStatement(t *testing.T) {
	digest := strings.Repeat("ab", sha256.Size)
	for name, data := range map[string]string{
		"no JSON":               `not json`,
		"other type":            `{"_type":"other","subject":[{"name":"page","digest":{"sha256":"` + digest + `"}}],"predicateType":"https://example.org/page"}`,
		"other predicate":       `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"page","digest":{"sha256":"` + digest + `"}}],"predicateType":"https://example.org/other"}`,
		"other subject":         `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"other","digest":{"sha256":"` + digest + `"}}],"predicateType":"https://example.org/page"}`,
		"two subjects":          `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"page","digest":{"sha256":"` + digest + `"}},{"name":"page","digest":{"sha256":"` + digest + `"}}],"predicateType":"https://example.org/page"}`,
		"no SHA-256 digest":     `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"page","digest":{"sha512":"` + digest + `"}}],"predicateType":"https://example.org/page"}`,
		"short digest":          `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"page","digest":{"sha256":"abab"}}],"predicateType":"https://example.org/page"}`,
		"digest that is no hex": `{"_type":"https://in-toto.io/Statement/v1","subject":[{"name":"page","digest":{"sha256":"zz"}}],"predicateType":"https://example.org/page"}`,
	} {
		if _, err := intotostatements.Unmarshal(
			[]byte(data),
			kind,
		); !errors.Is(
			err,
			intotostatements.ErrOtherStatement,
		) {
			t.Errorf("%s: err = %v, want ErrOtherStatement", name, err)
		}
	}
}
