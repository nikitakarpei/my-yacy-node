package wordjoined

type LeadingQueryWordChoice string

const (
	RarestSampledQueryWord        LeadingQueryWordChoice = "rarest sampled word"
	RarestQueryWordWithoutASample LeadingQueryWordChoice = "rarest word, no sample"
)

func leadingQueryWordChoiceOf(round discoveryRound) LeadingQueryWordChoice {
	if round.sampledLeadingQueryWord.Present() {
		return RarestSampledQueryWord
	}

	return RarestQueryWordWithoutASample
}
