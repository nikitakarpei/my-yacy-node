package provensearch

import (
	"encoding/binary"
	"math/bits"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

const (
	postingLength   = 6
	titleFlag       = 1
	impactPerHitBit = 32
	impactOfTitle   = 256
)

type Language byte

const AnyLanguage Language = 0

type Posting struct {
	Hits     uint16
	Day      uint16
	Language Language
	InTitle  bool
}

func PostingOf(posting Posting) []byte {
	value := binary.BigEndian.AppendUint16(nil, posting.Hits)
	value = binary.BigEndian.AppendUint16(value, posting.Day)
	value = append(value, byte(posting.Language))
	if posting.InTitle {
		return append(value, titleFlag)
	}

	return append(value, 0)
}

func PostingFrom(value []byte) (Posting, bool) {
	if len(value) != postingLength {
		return Posting{}, false
	}

	return Posting{
		Hits:     binary.BigEndian.Uint16(value[0:2]),
		Day:      binary.BigEndian.Uint16(value[2:4]),
		Language: Language(value[4]),
		InTitle:  value[5]&titleFlag != 0,
	}, true
}

func SummaryOf(value []byte) indextree.Summary {
	if posting, isPosting := PostingFrom(value); isPosting {
		return indextree.Summary{Postings: 1, TopImpact: ImpactOf(posting)}
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
