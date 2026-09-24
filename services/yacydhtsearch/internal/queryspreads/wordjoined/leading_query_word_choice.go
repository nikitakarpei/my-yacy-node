package wordjoined

type LeadingQueryWordChoice string

const (
	RarestSampledQueryWord                  LeadingQueryWordChoice = "rarest sampled word"
	RarestQueryWordWithoutCompleteAbstracts LeadingQueryWordChoice = "rarest word, partial abstracts"
)

func leadingQueryWordChoiceOf(round discoveryRound) LeadingQueryWordChoice {
	if round.samples.rarestQueryWord().Present() {
		return RarestSampledQueryWord
	}

	return RarestQueryWordWithoutCompleteAbstracts
}
