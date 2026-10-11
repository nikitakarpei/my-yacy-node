package probeanswerhistory

import (
	"fmt"
	"strconv"
	"strings"
)

type ProbeAnswerPosition struct {
	AppendedAtMillis uint64
	Sequence         uint64
}

func ProbeAnswerPositionOf(entryID string) (ProbeAnswerPosition, error) {
	millis, sequence, found := strings.Cut(entryID, "-")
	if !found {
		return ProbeAnswerPosition{}, fmt.Errorf("history position %q has no sequence", entryID)
	}
	appendedAtMillis, err := strconv.ParseUint(millis, 10, 64)
	if err != nil {
		return ProbeAnswerPosition{}, fmt.Errorf("history position %q: %w", entryID, err)
	}
	sequenceNumber, err := strconv.ParseUint(sequence, 10, 64)
	if err != nil {
		return ProbeAnswerPosition{}, fmt.Errorf("history position %q: %w", entryID, err)
	}

	return ProbeAnswerPosition{AppendedAtMillis: appendedAtMillis, Sequence: sequenceNumber}, nil
}

func EarliestOf(positions ...ProbeAnswerPosition) ProbeAnswerPosition {
	if len(positions) == 0 {
		return ProbeAnswerPosition{}
	}
	earliest := positions[0]
	for _, position := range positions {
		if position.before(earliest) {
			earliest = position
		}
	}

	return earliest
}

func (position ProbeAnswerPosition) before(other ProbeAnswerPosition) bool {
	if position.AppendedAtMillis != other.AppendedAtMillis {
		return position.AppendedAtMillis < other.AppendedAtMillis
	}

	return position.Sequence < other.Sequence
}

func (position ProbeAnswerPosition) IsZero() bool {
	return position == ProbeAnswerPosition{}
}

func (position ProbeAnswerPosition) String() string {
	return strconv.FormatUint(position.AppendedAtMillis, 10) + "-" +
		strconv.FormatUint(position.Sequence, 10)
}

func (position ProbeAnswerPosition) MarshalText() ([]byte, error) {
	return []byte(position.String()), nil
}

func (position *ProbeAnswerPosition) UnmarshalText(text []byte) error {
	parsed, err := ProbeAnswerPositionOf(string(text))
	if err != nil {
		return err
	}
	*position = parsed

	return nil
}
