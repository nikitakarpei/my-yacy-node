package yacyproto

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// calendarDayWireLayout matches YaCy's GenericFormatter.PATTERN_SHORT_DAY.
const calendarDayWireLayout = "20060102"

// calendarDayWireCodec translates between the calendar day domain type and the
// short day text YaCy carries in a url metadata row.
type calendarDayWireCodec struct{}

func (calendarDayWireCodec) encode(day yacymodel.CalendarDay) string {
	if day.IsZero() {
		return ""
	}

	return day.Time().Format(calendarDayWireLayout)
}

func (calendarDayWireCodec) decode(text string) yacymodel.CalendarDay {
	instant, err := time.Parse(calendarDayWireLayout, text)
	if err != nil {
		return yacymodel.CalendarDay{}
	}

	return yacymodel.CalendarDayOf(instant)
}
