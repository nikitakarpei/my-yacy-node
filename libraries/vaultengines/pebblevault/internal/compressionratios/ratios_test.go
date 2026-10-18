package compressionratios_test

import (
	"testing"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/sstable"

	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/internal/compressionratios"
)

const megabyte = 1 << 20

var words = compressionratios.BucketTally{Prefix: []byte("w"), Records: 100, HeldBytes: 1100}

const rawWordsBytes = 2000

func tableBetween(smallestKey, largestKey string, fileBytes, rawBytes uint64) pebble.SSTableInfo {
	info := pebble.SSTableInfo{Properties: &sstable.Properties{}}
	info.Smallest = pebble.InternalKey{UserKey: []byte(smallestKey)}
	info.Largest = pebble.InternalKey{UserKey: []byte(largestKey)}
	info.Size = fileBytes
	info.Properties.RawKeySize = rawBytes / 2
	info.Properties.RawValueSize = rawBytes - rawBytes/2

	return info
}

func TestBucketWithoutTablesTakesItsHeldBytes(t *testing.T) {
	if got := compressionratios.Of(nil).DiskBytesOf(words); got != words.HeldBytes {
		t.Fatalf("DiskBytesOf = %d, want the held bytes %d", got, words.HeldBytes)
	}
}

func TestBucketWithEnoughExclusiveTableBytesTakesTheirRatio(t *testing.T) {
	ratios := compressionratios.Of([][]pebble.SSTableInfo{
		{tableBetween("w1", "w5", 2*megabyte, 8*megabyte)},
		{
			tableBetween("w6", "w9", 2*megabyte, 8*megabyte),
			tableBetween("x1", "x9", megabyte, megabyte),
		},
	})

	if got := ratios.DiskBytesOf(words); got != rawWordsBytes/4 {
		t.Fatalf("DiskBytesOf = %d, want %d", got, rawWordsBytes/4)
	}
}

func TestBucketWithTooFewExclusiveTableBytesTakesTheRatioOfEveryTable(t *testing.T) {
	ratios := compressionratios.Of([][]pebble.SSTableInfo{{
		tableBetween("w1", "w9", megabyte, 4*megabyte),
		tableBetween("x1", "x9", 3*megabyte, 4*megabyte),
	}})

	if got := ratios.DiskBytesOf(words); got != rawWordsBytes/2 {
		t.Fatalf("DiskBytesOf = %d, want %d", got, rawWordsBytes/2)
	}
}

func TestTableAcrossTwoBucketsCountsForNeither(t *testing.T) {
	ratios := compressionratios.Of([][]pebble.SSTableInfo{{
		tableBetween("w1", "w8", 4*megabyte, 8*megabyte),
		tableBetween("w9", "x1", 4*megabyte, 4*megabyte),
	}})

	if got := ratios.DiskBytesOf(words); got != rawWordsBytes/2 {
		t.Fatalf("DiskBytesOf = %d, want %d", got, rawWordsBytes/2)
	}
}

func TestRatioStaysAboveItsLowestBound(t *testing.T) {
	ratios := compressionratios.Of([][]pebble.SSTableInfo{{tableBetween("x1", "x9", 1, megabyte)}})

	if got := ratios.DiskBytesOf(words); got != rawWordsBytes/20 {
		t.Fatalf("DiskBytesOf = %d, want %d", got, rawWordsBytes/20)
	}
}

func TestRatioStaysBelowItsHighestBound(t *testing.T) {
	ratios := compressionratios.Of(
		[][]pebble.SSTableInfo{{tableBetween("x1", "x9", 2*megabyte, megabyte)}},
	)

	if got := ratios.DiskBytesOf(words); got != rawWordsBytes*3/2 {
		t.Fatalf("DiskBytesOf = %d, want %d", got, rawWordsBytes*3/2)
	}
}
