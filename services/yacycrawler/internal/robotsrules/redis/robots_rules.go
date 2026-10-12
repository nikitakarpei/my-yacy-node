// Package redis keeps the robots.txt rules of every site for as long as they stay valid, and
// reads the rules it does not keep from the robots rules it wraps.
package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redisrecord"
)

const recordKind = "YACY_ROBOTS_RULES"

type RobotsRules struct {
	client   redis.UniversalClient
	inner    robotsexclusion.RobotsRules
	observer RobotsRulesObserver
}

func New(
	client redis.UniversalClient,
	inner robotsexclusion.RobotsRules,
	observer RobotsRulesObserver,
) *RobotsRules {
	return &RobotsRules{client: client, inner: inner, observer: observer}
}

func (kept *RobotsRules) RulesOf(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) robotsexclusion.Rules {
	if rules, found := kept.keptRulesOf(ctx, robotsTxtURL); found {
		kept.observer.RobotsRulesFound(ctx, robotsTxtURL)
		return rules
	}
	kept.observer.RobotsRulesMissed(ctx, robotsTxtURL)
	rules := kept.inner.RulesOf(ctx, robotsTxtURL)
	kept.keepRules(ctx, robotsTxtURL, rules)
	return rules
}

func (kept *RobotsRules) keptRulesOf(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) (robotsexclusion.Rules, bool) {
	rules, found, err := kept.readRules(ctx, robotsTxtURL)
	if err != nil {
		kept.observer.RobotsRulesNotRead(ctx, robotsTxtURL, err)
		return robotsexclusion.Rules{}, false
	}
	return rules, found
}

func (kept *RobotsRules) readRules(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) (robotsexclusion.Rules, bool, error) {
	payload, err := kept.client.Get(ctx, robotsRulesKeyOf(robotsTxtURL)).Bytes()
	if errors.Is(err, redis.Nil) {
		return robotsexclusion.Rules{}, false, nil
	}
	if err != nil {
		return robotsexclusion.Rules{}, false, fmt.Errorf(
			"get robots rules %s: %w", robotsTxtURL, err,
		)
	}
	rules, err := unmarshalRules(payload)
	if err != nil {
		return robotsexclusion.Rules{}, false, err
	}
	return rules, true, nil
}

func robotsRulesKeyOf(robotsTxtURL canonicalurl.CanonicalURL) string {
	return redisrecord.KeyOf(recordKind, robotsTxtURL.String())
}

func (kept *RobotsRules) keepRules(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	rules robotsexclusion.Rules,
) {
	if rules.Lifetime <= 0 {
		return
	}
	if err := kept.putRules(ctx, robotsTxtURL, rules); err != nil {
		kept.observer.RobotsRulesNotKept(ctx, robotsTxtURL, err)
	}
}

func (kept *RobotsRules) putRules(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	rules robotsexclusion.Rules,
) error {
	payload, err := marshalRules(rules)
	if err != nil {
		return err
	}
	err = kept.client.Set(ctx, robotsRulesKeyOf(robotsTxtURL), payload, rules.Lifetime).Err()
	if err != nil {
		return fmt.Errorf("put robots rules %s: %w", robotsTxtURL, err)
	}
	return nil
}
