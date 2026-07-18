package yacymodel

// microDateWireModulus matches YaCy's MicroDate.microDateDays: day counts wrap
// at 64**3 so they fit the row's fixed-width cardinal columns.
const microDateWireModulus = 262144

func (d MicroDate) WireDays() uint64 {
	days := int64(d) % microDateWireModulus
	if days < 0 {
		days += microDateWireModulus
	}

	return uint64(days) //nolint:gosec // wrapped above into [0, microDateWireModulus)
}

func MicroDateFromWireDays(days uint64) MicroDate {
	return MicroDate(days % microDateWireModulus)
}
