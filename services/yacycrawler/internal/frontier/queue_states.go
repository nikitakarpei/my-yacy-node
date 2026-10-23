package frontier

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	queueStatesBucket   = "crawl_queue_states"
	queuesInStateBucket = "crawl_queues_in_state"
)

var errQueueStateMalformed = errors.New("crawl queue state is malformed")

type queueStates struct {
	ofCrawl       *vault.Collection[crawlid.CrawlID, queueState]
	queuesInState *vault.ValueIndex[queueInState, queueInState]
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

func registerQueueStates(ctx context.Context, storage *vault.Vault) (queueStates, error) {
	var states queueStates
	var err error
	if states.queuesInState, err = storage.RegisterValueIndex(
		queuesInStateBucket, frontierIndexesVersion, queueInStateEntries,
		func(queue queueInState) queueInState { return queue },
	); err != nil {
		return queueStates{}, fmt.Errorf("register %s: %w", queuesInStateBucket, err)
	}
	if states.ofCrawl, err = storage.RegisterProjectedCollection(
		ctx,
		queueStatesBucket,
		vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout(),
		queueStateCodec{},
		vault.ProjectedAs(func(crawlID crawlid.CrawlID, state queueState) queueInState {
			return queueInState{state: int64(state), crawlID: crawlID}
		}, states.queuesInState),
	); err != nil {
		return queueStates{}, fmt.Errorf("register %s: %w", queueStatesBucket, err)
	}
	return states, nil
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
	err := states.queuesInState.Scan(
		tx,
		queueInStateKeyParts.KeysWithFirst(int64(queueDraining)),
		func(queue queueInState) (bool, error) {
			draining = append(draining, queue.crawlID)
			return len(draining) < amount, nil
		},
	)
	return draining, err
}

type queueInState struct {
	state   int64
	crawlID crawlid.CrawlID
}

var queueInStateKeyParts = vault.PairKey(vault.IntegerKeyPart, vaultkeyparts.CrawlID)

var queueInStateEntries = queueInStateKeyParts.KeyLayoutFor(
	func(queue queueInState) (int64, crawlid.CrawlID) { return queue.state, queue.crawlID },
	func(state int64, crawlID crawlid.CrawlID) queueInState {
		return queueInState{state: state, crawlID: crawlID}
	},
)

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
