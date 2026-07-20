package yacyproto

import (
	"fmt"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const utcOffsetWireLength = 5

// utcOffsetWireCodec translates between the UTC offset domain type and YaCy's
// signed SHHMM field: a sign character followed by two-digit hours and minutes.
type utcOffsetWireCodec struct{}

func (utcOffsetWireCodec) decode(text string) (yacymodel.UTCOffset, error) {
	if len(text) != utcOffsetWireLength {
		return yacymodel.UTCOffset{}, fmt.Errorf(
			"%w: utc offset %q",
			yacymodel.ErrBadUTCOffset,
			text,
		)
	}
	sign := 1
	switch text[0] {
	case '+':
	case '-':
		sign = -1
	default:
		return yacymodel.UTCOffset{}, fmt.Errorf("%w: sign %q", yacymodel.ErrBadUTCOffset, text[0])
	}
	hours, err := strconv.Atoi(text[1:3])
	if err != nil {
		return yacymodel.UTCOffset{}, fmt.Errorf("%w: %w", yacymodel.ErrBadUTCOffset, err)
	}
	minutes, err := strconv.Atoi(text[3:5])
	if err != nil {
		return yacymodel.UTCOffset{}, fmt.Errorf("%w: %w", yacymodel.ErrBadUTCOffset, err)
	}
	return yacymodel.NewUTCOffset(sign * (hours*60 + minutes))
}

func (utcOffsetWireCodec) encode(offset yacymodel.UTCOffset) string {
	minutes := offset.MinutesEast()
	sign := byte('+')
	if minutes < 0 {
		sign = '-'
		minutes = -minutes
	}
	return fmt.Sprintf("%c%02d%02d", sign, minutes/60, minutes%60)
}
