// Package replydeadlines holds when reading a page must end and when the
// response headers are due, from the arrival of the request.
package replydeadlines

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var preferWaitPattern = regexp.MustCompile(`(?:^|[\s,;])wait\s*=\s*(\d+)\b`)

type Timeouts struct {
	ResponseHeader   time.Duration
	AssessmentBudget time.Duration
}

type Deadlines struct {
	ReadingEndsAt time.Time
	HeadersDueAt  time.Time
}

func DeadlinesFrom(requestArrivedAt time.Time, preferences []string, timeouts Timeouts) Deadlines {
	responseHeaderTimeout := slices.Min(append(waitsOf(preferences), timeouts.ResponseHeader))
	headersDueAt := requestArrivedAt.Add(responseHeaderTimeout)
	return Deadlines{
		ReadingEndsAt: headersDueAt.Add(-timeouts.AssessmentBudget),
		HeadersDueAt:  headersDueAt,
	}
}

func waitsOf(preferences []string) []time.Duration {
	var waits []time.Duration
	for _, preference := range preferences {
		for _, match := range preferWaitPattern.FindAllStringSubmatch(strings.ToLower(preference), -1) {
			if seconds, err := strconv.ParseInt(match[1], 10, 32); err == nil {
				waits = append(waits, time.Duration(seconds)*time.Second)
			}
		}
	}
	return waits
}
