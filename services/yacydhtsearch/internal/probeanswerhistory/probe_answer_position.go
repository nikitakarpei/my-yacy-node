package probeanswerhistory

type ProbeAnswerPosition uint64

func EarliestOf(positions ...ProbeAnswerPosition) ProbeAnswerPosition {
	if len(positions) == 0 {
		return 0
	}
	earliest := positions[0]
	for _, position := range positions {
		earliest = min(earliest, position)
	}

	return earliest
}

func (position ProbeAnswerPosition) next() ProbeAnswerPosition {
	return position + 1
}
