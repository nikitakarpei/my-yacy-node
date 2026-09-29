// Package replydeadline holds when the response headers are due, from the
// arrival of the request.
package replydeadline

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var preferWaitPattern = regexp.MustCompile(`(?:^|[\s,;])wait\s*=\s*(\d+)\b`)

func HeadersDueAtFrom(
	requestArrivedAt time.Time,
	preferences []string,
	responseHeaderTimeout time.Duration,
) time.Time {
	return requestArrivedAt.Add(slices.Min(append(waitsOf(preferences), responseHeaderTimeout)))
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
