package optional_test

import (
	"encoding/json"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/optional"
)

func TestSomeHoldsItsValue(t *testing.T) {
	value, present := optional.Some(7).Get()

	if !present || value != 7 || optional.Some(7).OrElse(1) != 7 || !optional.Some(7).Present() {
		t.Errorf("Some(7) = %d, %v", value, present)
	}
}

func TestNoneHoldsNoValue(t *testing.T) {
	_, present := optional.None[int]().Get()

	if present || optional.None[int]().Present() || optional.None[int]().OrElse(1) != 1 {
		t.Error("None holds a value")
	}
}

func TestAValueRoundTripsAsJSON(t *testing.T) {
	for name, value := range map[string]optional.Value[string]{
		"some": optional.Some("text"),
		"none": optional.None[string](),
	} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var read optional.Value[string]
		if err := json.Unmarshal(data, &read); err != nil {
			t.Fatal(err)
		}
		if read != value {
			t.Errorf("%s: read %+v from %s, want %+v", name, read, data, value)
		}
	}
}

func TestAnAbsentValueIsNullInJSON(t *testing.T) {
	data, err := json.Marshal(optional.None[string]())
	if err != nil || string(data) != "null" {
		t.Errorf("None as JSON = %s, %v", data, err)
	}
}

func TestUnmarshalRefusesAValueOfAnotherType(t *testing.T) {
	var read optional.Value[int]
	if err := json.Unmarshal([]byte(`"text"`), &read); err == nil {
		t.Error("text unmarshals as an optional number")
	}
}

func TestMarshalReportsAValueThatHasNoJSON(t *testing.T) {
	if _, err := json.Marshal(optional.Some(func() {})); err == nil {
		t.Error("a function marshals as JSON")
	}
}
