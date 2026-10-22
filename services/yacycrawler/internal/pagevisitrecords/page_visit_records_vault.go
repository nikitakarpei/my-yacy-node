package pagevisitrecords

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const pageVisitsBucket = "page_visit_records"

var errLastPageVisitMalformed = errors.New("last page visit record has a malformed time")

func registerPageVisitRecords(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
) (*vault.ExpiringCollection[urlhash.URLHash, lastPageVisit], error) {
	lastVisits, err := storage.RegisterExpiringCollection(
		ctx,
		pageVisitsBucket,
		pageKeys,
		lastPageVisitCodec{},
		keptFor,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", pageVisitsBucket, err)
	}
	return lastVisits, nil
}

var pageKeys = vault.SingleKey(vaultkeyparts.URLHash).KeyLayout()

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
