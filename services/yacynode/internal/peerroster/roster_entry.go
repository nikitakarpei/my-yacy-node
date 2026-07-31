package peerroster

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	timestampWidth   = 8
	entryTimestamps  = 2
	entryHeaderWidth = timestampWidth * entryTimestamps
)

type rosterEntry struct {
	seed          yacymodel.Seed
	lastReachable time.Time
	lastContacted time.Time
}

type rosterEntryCodec struct{}

func (rosterEntryCodec) Encode(entry rosterEntry) ([]byte, error) {
	seed := yacyproto.EncodeSeed(entry.seed)
	out := make([]byte, entryHeaderWidth, entryHeaderWidth+len(seed))
	binary.BigEndian.PutUint64(out[:timestampWidth], uint64(entry.lastReachable.UnixNano()))
	binary.BigEndian.PutUint64(out[timestampWidth:], uint64(entry.lastContacted.UnixNano()))

	return append(out, seed...), nil
}

func (rosterEntryCodec) Decode(raw []byte) (rosterEntry, error) {
	if len(raw) < entryHeaderWidth {
		return rosterEntry{}, fmt.Errorf("decode roster entry: short record")
	}

	//nolint:gosec // round-trips an int64 UnixNano stored as fixed-width bytes
	lastReachableNanos := int64(binary.BigEndian.Uint64(raw[:timestampWidth]))
	//nolint:gosec // round-trips an int64 UnixNano stored as fixed-width bytes
	lastContactedNanos := int64(binary.BigEndian.Uint64(raw[timestampWidth:entryHeaderWidth]))
	seed, err := yacyproto.ParseSeed(context.Background(), string(raw[entryHeaderWidth:]))
	if err != nil {
		return rosterEntry{}, fmt.Errorf("decode roster entry: %w", err)
	}

	return rosterEntry{
		seed:          seed,
		lastReachable: time.Unix(0, lastReachableNanos),
		lastContacted: time.Unix(0, lastContactedNanos),
	}, nil
}
