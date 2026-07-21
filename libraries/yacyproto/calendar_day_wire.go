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

func (calendarDayWireCodec) encode(day yacymodel.Optional[yacymodel.CalendarDay]) string {
	value, ok := day.Get()
	if !ok {
		return ""
	}

	return value.Time().Format(calendarDayWireLayout)
}

func (calendarDayWireCodec) decode(text string) yacymodel.Optional[yacymodel.CalendarDay] {
	instant, err := time.Parse(calendarDayWireLayout, text)
	if err != nil {
		return yacymodel.None[yacymodel.CalendarDay]()
	}

	return yacymodel.Some(yacymodel.CalendarDayOf(instant))
}
