package origin

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

var (
	errSiteAskedToWait     = errors.New("the site asked to wait")
	errFetchDeadlinePassed = errors.New("the fetch deadline passed")
	errUnexpectedAnswer    = errors.New(
		"the site answered not modified to a fetch without a version",
	)
)

func unreachableCauseOf(fetchOutcome pagefetch.FetchOutcome) error {
	switch {
	case fetchOutcome.FailureCause != nil:
		return fetchOutcome.FailureCause
	case fetchOutcome.Status == pagefetch.FetchDeferred:
		return fmt.Errorf("%w %v", errSiteAskedToWait, fetchOutcome.DeferFor)
	case fetchOutcome.Status == pagefetch.FetchDeadlinePassed:
		return errFetchDeadlinePassed
	default:
		return errUnexpectedAnswer
	}
}
