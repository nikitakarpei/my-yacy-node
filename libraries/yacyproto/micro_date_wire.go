package yacyproto

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

const microDateWireModulus = uint16ColumnCeiling + 1

type microDateWireCodec struct{}

func (microDateWireCodec) encode(d yacymodel.MicroDate) uint64 {
	days := int64(d) % microDateWireModulus
	if days < 0 {
		days += microDateWireModulus
	}

	return uint64(days) //nolint:gosec // wrapped above into [0, microDateWireModulus)
}

func (microDateWireCodec) decode(days uint16) yacymodel.MicroDate {
	return yacymodel.MicroDate(days)
}
