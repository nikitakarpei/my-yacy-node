package indextree

import "encoding/binary"

type Summary struct {
	Live         uint64
	Removed      uint64
	TopImpact    uint16
	TopStaleness uint16
}

func (s Summary) combinedWith(other Summary) Summary {
	return Summary{
		Live:         s.Live + other.Live,
		Removed:      s.Removed + other.Removed,
		TopImpact:    max(s.TopImpact, other.TopImpact),
		TopStaleness: max(s.TopStaleness, other.TopStaleness),
	}
}

func (s Summary) appendBytes(out []byte) []byte {
	out = binary.BigEndian.AppendUint64(out, s.Live)
	out = binary.BigEndian.AppendUint64(out, s.Removed)
	out = binary.BigEndian.AppendUint16(out, s.TopImpact)

	return binary.BigEndian.AppendUint16(out, s.TopStaleness)
}

const summaryLength = 20

type SummaryRule func(value []byte) Summary
