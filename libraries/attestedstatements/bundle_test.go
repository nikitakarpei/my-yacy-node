package attestedstatements_test

import (
	"encoding/json"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
)

type bundleCarrier struct {
	Bundle attestedstatements.Bundle `json:"bundle,omitzero"`
}

func TestABundleTravelsInJSONAsItIs(t *testing.T) {
	encoding, err := json.Marshal(
		bundleCarrier{Bundle: attestedstatements.Bundle(`{"mediaType":"m"}`)},
	)
	if err != nil || string(encoding) != `{"bundle":{"mediaType":"m"}}` {
		t.Fatalf("encoding = %s, %v", encoding, err)
	}
	var carrier bundleCarrier
	if err := json.Unmarshal(
		encoding,
		&carrier,
	); err != nil ||
		string(carrier.Bundle) != `{"mediaType":"m"}` {
		t.Fatalf("bundle = %s, %v", carrier.Bundle, err)
	}
}

func TestNoBundleIsLeftOutOfJSON(t *testing.T) {
	encoding, err := json.Marshal(bundleCarrier{})
	if err != nil || string(encoding) != `{}` {
		t.Fatalf("encoding = %s, %v", encoding, err)
	}
}

func TestAnEmptyBundleIsNull(t *testing.T) {
	encoding, err := attestedstatements.Bundle(nil).MarshalJSON()
	if err != nil || string(encoding) != "null" {
		t.Fatalf("encoding = %s, %v", encoding, err)
	}
}
