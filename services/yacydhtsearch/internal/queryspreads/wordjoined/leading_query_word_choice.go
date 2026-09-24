package wordjoined

type LeadingQueryWordChoice string

const (
	RarestQueryWordWithASample    LeadingQueryWordChoice = "rarest word with a sample"
	RarestQueryWordWithoutASample LeadingQueryWordChoice = "rarest word, no sample"
)

func leadingQueryWordChoiceOf(round discoveryRound) LeadingQueryWordChoice {
	if round.leadingQueryWordFromTheSample.Present() {
		return RarestQueryWordWithASample
	}

	return RarestQueryWordWithoutASample
}
