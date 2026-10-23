package frontier

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const queueStatesBucket = "crawl_queue_states"

var errQueueStateMalformed = errors.New("crawl queue state is malformed")

type queueStates struct {
	ofCrawl *vault.Collection[crawlid.CrawlID, queueState]
}

type queueState int64

const (
	queueOpen queueState = iota
	queuePaused
	queueDraining
)

func (state queueState) takesPageVisits() bool {
	return state != queueDraining
}

func registerQueueStates(storage *vault.Vault) (queueStates, error) {
	ofCrawl, err := storage.RegisterCollection(
		queueStatesBucket, vault.SingleKey(crawlid.KeyPart).KeyLayout(), queueStateCodec{},
	)
	if err != nil {
		return queueStates{}, fmt.Errorf("register %s: %w", queueStatesBucket, err)
	}
	return queueStates{ofCrawl: ofCrawl}, nil
}

func (states queueStates) stateOf(tx *vault.Txn, crawlID crawlid.CrawlID) (queueState, error) {
	state, _, err := states.ofCrawl.Get(tx, crawlID)
	return state, err
}

func (states queueStates) put(tx *vault.Txn, crawlID crawlid.CrawlID, state queueState) error {
	_, err := states.ofCrawl.Put(tx, crawlID, state)
	return err
}

func (states queueStates) remove(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	_, err := states.ofCrawl.Delete(tx, crawlID)
	return err
}

func (states queueStates) firstDrainingQueues(
	tx *vault.Txn,
	amount int,
) ([]crawlid.CrawlID, error) {
	var draining []crawlid.CrawlID
	err := states.ofCrawl.Scan(
		tx,
		vault.EveryKey(),
		func(crawlID crawlid.CrawlID, state queueState) (bool, error) {
			if state == queueDraining {
				draining = append(draining, crawlID)
			}
			return len(draining) < amount, nil
		},
	)
	return draining, err
}

type queueStateCodec struct{}

func (queueStateCodec) Encode(state queueState) ([]byte, error) {
	return binary.AppendVarint(nil, int64(state)), nil
}

func (queueStateCodec) Decode(stored []byte) (queueState, error) {
	state, read := binary.Varint(stored)
	if read <= 0 {
		return queueOpen, fmt.Errorf("%w: %d bytes", errQueueStateMalformed, len(stored))
	}
	return queueState(state), nil
}
