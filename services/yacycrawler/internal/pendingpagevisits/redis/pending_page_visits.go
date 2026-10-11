package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redisrecord"
)

const (
	frontierName    = "YACY_CRAWL_FRONTIER"
	dueTimesKey     = "{" + frontierName + "}:due-times"
	pageVisitsKey   = "{" + frontierName + "}:page-visits"
	lastSequenceKey = "{" + frontierName + "}:last-sequence"
	pageVisitKind   = "YACY_PENDING_PAGE_VISIT"
	noDueVisitPause = 500 * time.Millisecond
	entryBoundary   = " "
)

var addPendingPageVisit = redis.NewScript(`
if redis.call('HEXISTS', KEYS[2], ARGV[1]) == 1 then return 0 end
local sequence = redis.call('INCR', KEYS[3])
redis.call('HSET', KEYS[2], ARGV[1], sequence .. ' ' .. ARGV[3])
redis.call('ZADD', KEYS[1], ARGV[2], ARGV[1])
return sequence
`)

var leaseDuePageVisit = redis.NewScript(`
local fields = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, 1)
if #fields == 0 then return false end
redis.call('ZADD', KEYS[1], ARGV[2], fields[1])
return {fields[1], redis.call('HGET', KEYS[2], fields[1])}
`)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type FrontierObserver interface {
	FrontierLeaseFailed(ctx context.Context, err error)
	FrontierCountFailed(ctx context.Context, err error)
	FrontierAcknowledgeFailed(ctx context.Context, identity string, err error)
	FrontierReturnFailed(
		ctx context.Context,
		identity string,
		delay time.Duration,
		err error,
	)
}

type PendingPageVisits struct {
	client   redis.UniversalClient
	lease    time.Duration
	clock    Clock
	observer FrontierObserver
}

func New(
	client redis.UniversalClient,
	lease time.Duration,
	clock Clock,
	observer FrontierObserver,
) *PendingPageVisits {
	return &PendingPageVisits{client: client, lease: lease, clock: clock, observer: observer}
}

func (visits *PendingPageVisits) Publish(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
) error {
	body, err := pendingpagevisit.MarshalPendingPageVisit(pageVisit)
	if err != nil {
		return err
	}
	now := visits.clock.Now()
	if err := addPendingPageVisit.Run(ctx, visits.client,
		[]string{dueTimesKey, pageVisitsKey, lastSequenceKey},
		fieldOf(pageVisit), now.UnixMilli(), arrivalOf(now)+entryBoundary+string(body),
	).Err(); err != nil {
		return fmt.Errorf("add pending page visit %s: %w", pageVisit.URL, err)
	}
	return nil
}

func arrivalOf(now time.Time) string {
	return strconv.FormatInt(now.UnixMilli(), 10)
}

func fieldOf(pageVisit pendingpagevisit.PendingPageVisit) string {
	return redisrecord.KeyOf(pageVisitKind, pageVisit.OrderID, pageVisit.URL.String())
}

func (visits *PendingPageVisits) Next(ctx context.Context) (pullintake.PendingMessage, error) {
	for {
		message, leased := visits.leasedPageVisit(ctx)
		if leased {
			return message, nil
		}
		if err := visits.clock.Sleep(ctx, noDueVisitPause); err != nil {
			return nil, err
		}
	}
}

func (visits *PendingPageVisits) leasedPageVisit(
	ctx context.Context,
) (pendingPageVisitMessage, bool) {
	now := visits.clock.Now()
	reply, err := leaseDuePageVisit.Run(ctx, visits.client,
		[]string{dueTimesKey, pageVisitsKey},
		now.UnixMilli(), now.Add(visits.lease).UnixMilli(),
	).StringSlice()
	if errors.Is(err, redis.Nil) {
		return pendingPageVisitMessage{}, false
	}
	if err != nil {
		visits.observer.FrontierLeaseFailed(ctx, err)
		return pendingPageVisitMessage{}, false
	}
	message, err := visits.messageOf(reply[0], reply[1])
	if err != nil {
		visits.observer.FrontierLeaseFailed(ctx, err)
		return pendingPageVisitMessage{}, false
	}
	return message, true
}

func (visits *PendingPageVisits) messageOf(
	field string,
	entry string,
) (pendingPageVisitMessage, error) {
	sequence, rest, _ := strings.Cut(entry, entryBoundary)
	arrival, body, _ := strings.Cut(rest, entryBoundary)
	arrivalMilli, err := strconv.ParseInt(arrival, 10, 64)
	if err != nil {
		return pendingPageVisitMessage{}, fmt.Errorf(
			"read the arrival of pending page visit %s: %w",
			field,
			err,
		)
	}
	return pendingPageVisitMessage{
		visits:   visits,
		field:    field,
		identity: frontierName + "/" + sequence,
		arrival:  time.UnixMilli(arrivalMilli),
		body:     []byte(body),
	}, nil
}

func (visits *PendingPageVisits) AmountOfPendingPageVisits(ctx context.Context) (int64, bool) {
	amount, err := visits.client.HLen(ctx, pageVisitsKey).Result()
	if err != nil {
		visits.observer.FrontierCountFailed(ctx, err)
		return 0, false
	}
	return amount, true
}

func (visits *PendingPageVisits) acknowledge(ctx context.Context, field string, identity string) {
	_, err := visits.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZRem(ctx, dueTimesKey, field)
		pipe.HDel(ctx, pageVisitsKey, field)
		return nil
	})
	if err != nil {
		visits.observer.FrontierAcknowledgeFailed(ctx, identity, err)
	}
}

func (visits *PendingPageVisits) returnAfter(
	ctx context.Context,
	field string,
	identity string,
	delay time.Duration,
) {
	dueTime := visits.clock.Now().Add(delay).UnixMilli()
	err := visits.client.ZAddXX(ctx, dueTimesKey, redis.Z{Score: float64(dueTime), Member: field}).
		Err()
	if err != nil {
		visits.observer.FrontierReturnFailed(ctx, identity, delay, err)
	}
}
