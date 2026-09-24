package wordjoined

type LeadingQueryWordChoice string

const (
	RarestQueryWordWithASample    LeadingQueryWordChoice = "rarest word with a sample"
	RarestQueryWordWithoutASample LeadingQueryWordChoice = "rarest word, no sample"
	RarestQueryWordRemembered     LeadingQueryWordChoice = "rarest word, remembered"
)

func leadingQueryWordChoiceOf(round discoveryRound) LeadingQueryWordChoice {
	if round.leadingQueryWordRemembered {
		return RarestQueryWordRemembered
	}
	if round.chosenLeadingQueryWord.Present() {
		return RarestQueryWordWithASample
	}

	return RarestQueryWordWithoutASample
}
