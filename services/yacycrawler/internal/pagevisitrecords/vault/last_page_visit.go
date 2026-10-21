package vault

import (
	"encoding/binary"
	"errors"
	"time"
)

var errLastPageVisitMalformed = errors.New("last page visit record has a malformed time")

type lastPageVisit struct {
	visitedAt  time.Time
	entityTag  string
	modifiedAt time.Time
}

type lastPageVisitCodec struct{}

func (lastPageVisitCodec) Encode(record lastPageVisit) ([]byte, error) {
	encoded := make([]byte, 0, 2*binary.MaxVarintLen64+len(record.entityTag))
	encoded = binary.AppendVarint(encoded, record.visitedAt.UnixMilli())
	encoded = binary.AppendVarint(encoded, modifiedAtMillisOf(record.modifiedAt))
	return append(encoded, record.entityTag...), nil
}

func modifiedAtMillisOf(modifiedAt time.Time) int64 {
	if modifiedAt.IsZero() {
		return 0
	}
	return modifiedAt.UnixMilli()
}

func (lastPageVisitCodec) Decode(stored []byte) (lastPageVisit, error) {
	visitedAt, visitedAtBytes := binary.Varint(stored)
	if visitedAtBytes <= 0 {
		return lastPageVisit{}, errLastPageVisitMalformed
	}
	modifiedAt, modifiedAtBytes := binary.Varint(stored[visitedAtBytes:])
	if modifiedAtBytes <= 0 {
		return lastPageVisit{}, errLastPageVisitMalformed
	}
	return lastPageVisit{
		visitedAt:  time.UnixMilli(visitedAt).UTC(),
		entityTag:  string(stored[visitedAtBytes+modifiedAtBytes:]),
		modifiedAt: modifiedAtFrom(modifiedAt),
	}, nil
}

func modifiedAtFrom(millis int64) time.Time {
	if millis == 0 {
		return time.Time{}
	}
	return time.UnixMilli(millis).UTC()
}
