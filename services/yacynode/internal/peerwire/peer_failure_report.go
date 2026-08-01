package peerwire

import (
	"io"
	"strings"
)

const (
	failureReportMaxBytes int64 = 512
	failureReportAbsent         = "no reported body"
)

func peerFailureReport(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, failureReportMaxBytes))
	if err != nil && len(raw) == 0 {
		return failureReportAbsent
	}

	report := strings.Join(strings.Fields(string(raw)), " ")
	if report == "" {
		return failureReportAbsent
	}

	return report
}
