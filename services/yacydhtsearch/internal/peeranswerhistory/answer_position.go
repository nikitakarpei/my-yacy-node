package peeranswerhistory

type AnswerPosition uint64

func EarliestOf(positions ...AnswerPosition) AnswerPosition {
	if len(positions) == 0 {
		return 0
	}
	earliest := positions[0]
	for _, position := range positions {
		earliest = min(earliest, position)
	}

	return earliest
}

func (position AnswerPosition) next() AnswerPosition {
	return position + 1
}
