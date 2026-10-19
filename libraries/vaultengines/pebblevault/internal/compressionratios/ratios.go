// Package compressionratios estimates the bytes a bucket takes on disk from the
// compression its Pebble tables reached. A bucket whose exclusive tables hold too
// few bytes takes the compression of every table. Without tables, a bucket takes
// its held bytes.
package compressionratios

import (
	"bytes"

	"github.com/cockroachdb/pebble/v2"
)

const (
	exclusiveTablesMinimumBytes = 4 << 20
	lowestRatio                 = 0.05
	highestRatio                = 1.5
)

type BucketTally struct {
	Prefix    []byte
	Records   int
	HeldBytes int64
}

type Ratios struct {
	tables     []table
	everyTable tableBytes
}

type table struct {
	smallestKey []byte
	largestKey  []byte
	bytes       tableBytes
}

type tableBytes struct {
	file uint64
	raw  uint64
}

func Of(tablesByLevel [][]pebble.SSTableInfo) Ratios {
	var ratios Ratios
	for _, level := range tablesByLevel {
		for _, info := range level {
			surveyedTable := tableOf(info)
			ratios.tables = append(ratios.tables, surveyedTable)
			ratios.everyTable = ratios.everyTable.plus(surveyedTable.bytes)
		}
	}

	return ratios
}

func tableOf(info pebble.SSTableInfo) table {
	return table{
		smallestKey: info.Smallest.UserKey,
		largestKey:  info.Largest.UserKey,
		bytes: tableBytes{
			file: info.Size,
			raw:  info.Properties.RawKeySize + info.Properties.RawValueSize,
		},
	}
}

func (b tableBytes) plus(other tableBytes) tableBytes {
	return tableBytes{file: b.file + other.file, raw: b.raw + other.raw}
}

func (r Ratios) DiskBytesOf(tally BucketTally) int64 {
	if r.everyTable.raw == 0 {
		return tally.HeldBytes
	}

	return int64(float64(rawBytesOf(tally)) * r.ratioFor(tally.Prefix))
}

func rawBytesOf(tally BucketTally) int64 {
	untalliedKeyBytes := pebble.InternalKey{UserKey: tally.Prefix}.Size()

	return tally.HeldBytes + int64(tally.Records*untalliedKeyBytes)
}

func (r Ratios) ratioFor(prefix []byte) float64 {
	exclusiveTableBytes := r.exclusiveTableBytesOf(prefix)
	if exclusiveTableBytes.file >= exclusiveTablesMinimumBytes {
		return exclusiveTableBytes.ratio()
	}

	return r.everyTable.ratio()
}

func (r Ratios) exclusiveTableBytesOf(prefix []byte) tableBytes {
	var exclusiveTableBytes tableBytes
	for _, surveyedTable := range r.tables {
		if surveyedTable.liesWithin(prefix) {
			exclusiveTableBytes = exclusiveTableBytes.plus(surveyedTable.bytes)
		}
	}

	return exclusiveTableBytes
}

func (t table) liesWithin(prefix []byte) bool {
	return bytes.HasPrefix(t.smallestKey, prefix) && bytes.HasPrefix(t.largestKey, prefix)
}

func (b tableBytes) ratio() float64 {
	return min(max(float64(b.file)/float64(b.raw), lowestRatio), highestRatio)
}
