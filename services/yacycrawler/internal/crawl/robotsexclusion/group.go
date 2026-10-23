package robotsexclusion

import (
	"time"

	"github.com/jimsmart/grobotstxt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const undeterminedDeferral = time.Minute

type Group struct {
	rules        Rules
	productToken ProductToken
}

func (group Group) FetchPermissionOf(pageURL canonicalurl.CanonicalURL) FetchPermission {
	switch group.rules.Availability {
	case Stated:
		return permissionOfStatedRules(grobotstxt.AgentAllowed(
			group.rules.RobotsTxt, string(group.productToken), pageURL.String(),
		))
	case Absent:
		return FetchPermission{Verdict: Allowed}
	case Unreachable:
		return FetchPermission{Verdict: Deferred, DeferFor: group.rules.Lifetime}
	default:
		return FetchPermission{Verdict: Deferred, DeferFor: undeterminedDeferral}
	}
}

func permissionOfStatedRules(allowed bool) FetchPermission {
	if allowed {
		return FetchPermission{Verdict: Allowed}
	}
	return FetchPermission{Verdict: Disallowed}
}
