package origin

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
)

const (
	answeredRulesLifetime   = 24 * time.Hour
	unreachableRulesFloor   = time.Minute
	unreachableRulesCeiling = time.Hour
)

func rulesFromFetch(fetchOutcome pagefetch.FetchOutcome) robotsexclusion.Rules {
	switch fetchOutcome.Status {
	case pagefetch.FetchSucceeded:
		return robotsexclusion.Rules{
			Availability: robotsexclusion.Stated,
			RobotsTxt:    string(fetchOutcome.Page.Body),
			Lifetime:     answeredRulesLifetime,
		}
	case pagefetch.FetchAccessRefused,
		pagefetch.FetchGone,
		pagefetch.FetchRejected,
		pagefetch.FetchOversized,
		pagefetch.FetchRedirected,
		pagefetch.FetchRedirectTargetInvalid:
		return robotsexclusion.Rules{
			Availability: robotsexclusion.Absent,
			Lifetime:     answeredRulesLifetime,
		}
	case pagefetch.FetchDeferred:
		return unreachableRulesFor(fetchOutcome.DeferFor)
	default:
		return unreachableRulesFor(unreachableRulesFloor)
	}
}

func unreachableRulesFor(wait time.Duration) robotsexclusion.Rules {
	return robotsexclusion.Rules{
		Availability: robotsexclusion.Unreachable,
		Lifetime:     min(max(wait, unreachableRulesFloor), unreachableRulesCeiling),
	}
}
