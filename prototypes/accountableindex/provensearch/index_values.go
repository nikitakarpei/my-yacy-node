package provensearch

import (
	"encoding/binary"
	"math/bits"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

const (
	postingKind          = 1
	urlTombstoneKind     = 2
	wordTombstoneKind    = 3
	postingLength        = 6
	urlTombstoneLength   = 10
	titleFlag            = 1
	impactPerHitBit      = 32
	impactOfTitle        = 256
	wordTombstoneMinimum = 9
)

func RemovedURLKeyOf(url indextree.URL) indextree.Key {
	return indextree.KeyOfKind(indextree.URLTombstoneKind, indextree.Word{}, url)
}

const amountOfPartitionBits = 4

func PartitionOf(url indextree.URL) byte {
	return url[0] >> (8 - amountOfPartitionBits)
}

func HandedOffPartitionKeyOf(word indextree.Word, partition byte) indextree.Key {
	return indextree.KeyOfKind(indextree.HandoffTombstoneKind, word, indextree.URL{partition})
}

type RemovalReason byte

const (
	RemovedByEviction RemovalReason = iota + 1
	RemovedByNewerVersion
	RemovedByOperator
	RemovedByReset
)

type Posting struct {
	Hits    uint16
	Day     uint16
	InTitle bool
}

func PostingOf(posting Posting) []byte {
	value := binary.BigEndian.AppendUint16([]byte{postingKind}, posting.Hits)
	value = binary.BigEndian.AppendUint16(value, posting.Day)
	if posting.InTitle {
		return append(value, titleFlag)
	}

	return append(value, 0)
}

func PostingFrom(value []byte) (Posting, bool) {
	if len(value) != postingLength || value[0] != postingKind {
		return Posting{}, false
	}

	return Posting{
		Hits:    binary.BigEndian.Uint16(value[1:3]),
		Day:     binary.BigEndian.Uint16(value[3:5]),
		InTitle: value[5]&titleFlag != 0,
	}, true
}

type URLTombstone struct {
	Epoch  uint64
	Reason RemovalReason
}

func URLTombstoneOf(tombstone URLTombstone) []byte {
	return binary.BigEndian.AppendUint64([]byte{urlTombstoneKind, byte(tombstone.Reason)}, tombstone.Epoch)
}

func URLTombstoneFrom(value []byte) (URLTombstone, bool) {
	if len(value) != urlTombstoneLength || value[0] != urlTombstoneKind {
		return URLTombstone{}, false
	}

	return URLTombstone{Reason: RemovalReason(value[1]), Epoch: binary.BigEndian.Uint64(value[2:10])}, true
}

type WordTombstone struct {
	Receivers []string
	Epoch     uint64
}

func WordTombstoneOf(tombstone WordTombstone) []byte {
	value := binary.BigEndian.AppendUint64([]byte{wordTombstoneKind}, tombstone.Epoch)
	for _, receiver := range tombstone.Receivers {
		value = append(value, byte(len(receiver)))
		value = append(value, receiver...)
	}

	return value
}

func WordTombstoneFrom(value []byte) (WordTombstone, bool) {
	if len(value) < wordTombstoneMinimum || value[0] != wordTombstoneKind {
		return WordTombstone{}, false
	}
	tombstone := WordTombstone{Epoch: binary.BigEndian.Uint64(value[1:9])}
	for rest := value[9:]; len(rest) > 0; {
		length := int(rest[0])
		if len(rest) < 1+length {
			return WordTombstone{}, false
		}
		tombstone.Receivers = append(tombstone.Receivers, string(rest[1:1+length]))
		rest = rest[1+length:]
	}

	return tombstone, true
}

func SummaryOf(value []byte) indextree.Summary {
	if posting, isPosting := PostingFrom(value); isPosting {
		return indextree.Summary{Live: 1, TopImpact: ImpactOf(posting)}
	}
	if _, isTombstone := URLTombstoneFrom(value); isTombstone {
		return indextree.Summary{Removed: 1}
	}
	if _, isTombstone := WordTombstoneFrom(value); isTombstone {
		return indextree.Summary{Removed: 1}
	}

	return indextree.Summary{}
}

func ImpactOf(posting Posting) uint16 {
	impact := uint16(impactPerHitBit * bits.Len16(posting.Hits))
	if posting.InTitle {
		impact += impactOfTitle
	}

	return impact
}
