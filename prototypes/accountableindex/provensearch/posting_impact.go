package provensearch

import (
	"encoding/binary"
	"math/bits"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

const (
	postingKind     = 1
	tombstoneKind   = 2
	postingLength   = 4
	tombstoneLength = 10
	titleFlag       = 1
	impactPerHitBit = 32
	impactOfTitle   = 256
)

type RemovalReason byte

const (
	RemovedByEviction RemovalReason = iota + 1
	RemovedByHandoff
	RemovedByNewerVersion
	RemovedByOperator
)

func PostingOf(hits uint16, inTitle bool) []byte {
	posting := binary.BigEndian.AppendUint16([]byte{postingKind}, hits)
	if inTitle {
		return append(posting, titleFlag)
	}

	return append(posting, 0)
}

func TombstoneOf(reason RemovalReason, epoch uint64) []byte {
	return binary.BigEndian.AppendUint64([]byte{tombstoneKind, byte(reason)}, epoch)
}

func TombstoneEpochOf(value []byte) (uint64, bool) {
	if len(value) != tombstoneLength || value[0] != tombstoneKind {
		return 0, false
	}

	return binary.BigEndian.Uint64(value[2:]), true
}

func SummaryOf(value []byte) indextree.Summary {
	switch {
	case len(value) == postingLength && value[0] == postingKind:
		return indextree.Summary{Live: 1, TopImpact: impactOf(value)}
	case len(value) == tombstoneLength && value[0] == tombstoneKind:
		return indextree.Summary{Removed: 1}
	default:
		return indextree.Summary{}
	}
}

func impactOf(posting []byte) uint16 {
	impact := uint16(impactPerHitBit * bits.Len16(binary.BigEndian.Uint16(posting[1:3])))
	if posting[3]&titleFlag != 0 {
		impact += impactOfTitle
	}

	return impact
}
