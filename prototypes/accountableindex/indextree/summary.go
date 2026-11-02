package indextree

import "encoding/binary"

type Summary struct {
	Postings  uint64
	TopImpact uint16
}

func (s Summary) combinedWith(other Summary) Summary {
	return Summary{
		Postings:  s.Postings + other.Postings,
		TopImpact: max(s.TopImpact, other.TopImpact),
	}
}

func (s Summary) appendBytes(out []byte) []byte {
	out = binary.BigEndian.AppendUint64(out, s.Postings)

	return binary.BigEndian.AppendUint16(out, s.TopImpact)
}

const summaryLength = 10

type SummaryRule func(value []byte) Summary
