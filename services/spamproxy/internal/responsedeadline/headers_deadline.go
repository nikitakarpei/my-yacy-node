// Package responsedeadline holds the response headers deadline, from the
// arrival of the request.
package responsedeadline

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var preferWaitPattern = regexp.MustCompile(`(?:^|[\s,;])wait\s*=\s*(\d+)\b`)

func HeadersDeadlineFrom(
	requestArrivedAt time.Time,
	preferHeaderValues []string,
	responseHeaderTimeout time.Duration,
) time.Time {
	return requestArrivedAt.Add(
		slices.Min(append(waitsOf(preferHeaderValues), responseHeaderTimeout)),
	)
}

func waitsOf(preferHeaderValues []string) []time.Duration {
	var waits []time.Duration
	for _, preferHeaderValue := range preferHeaderValues {
		for _, match := range preferWaitPattern.FindAllStringSubmatch(strings.ToLower(preferHeaderValue), -1) {
			if seconds, err := strconv.ParseInt(match[1], 10, 32); err == nil {
				waits = append(waits, time.Duration(seconds)*time.Second)
			}
		}
	}
	return waits
}
