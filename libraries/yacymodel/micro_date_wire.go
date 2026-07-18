package yacymodel

import (
	"fmt"
	"strconv"
)

// microDateWireModulus matches YaCy's MicroDate.microDateDays: day counts wrap
// at 64**3 so they fit the row's fixed-width cardinal columns.
const microDateWireModulus = 262144

func (d MicroDate) WireValue() string {
	days := int64(d) % microDateWireModulus
	if days < 0 {
		days += microDateWireModulus
	}
	return strconv.FormatInt(days, 10)
}

func ParseMicroDateWireValue(value string) (MicroDate, error) {
	days, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse micro date: %w", err)
	}
	return MicroDate(days % microDateWireModulus), nil
}
