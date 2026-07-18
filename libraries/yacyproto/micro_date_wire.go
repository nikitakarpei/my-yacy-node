package yacyproto

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

// microDateWireModulus matches YaCy's MicroDate.microDateDays: day counts wrap
// at 64**3 so they fit the row's fixed-width cardinal columns.
const microDateWireModulus = 262144

// microDateWireCodec translates between the micro date domain type and the day
// count YaCy carries in a posting's cardinal column.
type microDateWireCodec struct{}

func (microDateWireCodec) encode(d yacymodel.MicroDate) uint64 {
	days := int64(d) % microDateWireModulus
	if days < 0 {
		days += microDateWireModulus
	}

	return uint64(days) //nolint:gosec // wrapped above into [0, microDateWireModulus)
}

func (microDateWireCodec) decode(days uint64) yacymodel.MicroDate {
	return yacymodel.MicroDate(days % microDateWireModulus)
}
