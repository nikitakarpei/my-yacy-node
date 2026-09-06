package yacyproto

import "time"

// instantWireLayout matches YaCy's GenericFormatter.PATTERN_SHORT_SECOND,
// written in UTC.
const instantWireLayout = "20060102150405"

// instantWireCodec translates between an instant and the short-second text YaCy
// carries in its date fields.
type instantWireCodec struct{}

func (instantWireCodec) encode(instant time.Time) string {
	return instant.UTC().Format(instantWireLayout)
}

func (instantWireCodec) decode(text string) (time.Time, bool) {
	instant, err := time.ParseInLocation(instantWireLayout, text, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return instant, true
}
