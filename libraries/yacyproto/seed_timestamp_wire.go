package yacyproto

import "time"

// seedTimestampWireLayout matches YaCy's GenericFormatter.PATTERN_SHORT_SECOND,
// written in UTC.
const seedTimestampWireLayout = "20060102150405"

// seedTimestampWireCodec translates between an instant and the short-second text
// YaCy carries in a seed's date fields.
type seedTimestampWireCodec struct{}

func (seedTimestampWireCodec) encode(instant time.Time) string {
	return instant.UTC().Format(seedTimestampWireLayout)
}

func (seedTimestampWireCodec) decode(text string) (time.Time, bool) {
	instant, err := time.ParseInLocation(seedTimestampWireLayout, text, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return instant, true
}
