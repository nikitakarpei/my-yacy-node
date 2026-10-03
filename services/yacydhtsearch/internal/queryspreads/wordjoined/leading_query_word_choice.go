package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type chosenLeadingQueryWord struct {
	word   yacymodel.Optional[yacymodel.Hash]
	choice LeadingQueryWordChoice
}

type LeadingQueryWordChoice string

const (
	RarestQueryWordWithASample    LeadingQueryWordChoice = "rarest word with a sample"
	RarestQueryWordWithoutASample LeadingQueryWordChoice = "rarest word, no sample"
	RarestQueryWordRemembered     LeadingQueryWordChoice = "rarest word, remembered"
)
