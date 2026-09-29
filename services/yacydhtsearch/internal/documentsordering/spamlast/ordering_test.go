package spamlast_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/spamlast"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

type foundOrdering struct{}

func (foundOrdering) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return slices.Clone(answers.FoundDocuments)
}

type assessedAddress struct {
	address string
	verdict spamassessment.Verdict
}

func answersOf(assessedAddresses ...assessedAddress) queryanswers.AnsweredQuery {
	answers := queryanswers.AnsweredQuery{}
	for _, assessed := range assessedAddresses {
		answers.FoundDocuments = append(answers.FoundDocuments, queryanswers.FoundDocument{
			Address:     assessed.address,
			SpamVerdict: assessed.verdict,
		})
	}

	return answers
}

func addressesOf(foundDocuments []queryanswers.FoundDocument) []string {
	addresses := make([]string, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		addresses = append(addresses, foundDocument.Address)
	}

	return addresses
}

func TestSpamDocumentsComeAfterAllOthersInTheOrderOfTheOtherOrdering(t *testing.T) {
	t.Parallel()

	orderedDocuments := spamlast.New(foundOrdering{}).OrderedDocumentsOf(answersOf(
		assessedAddress{"https://first-spam.example/", spamassessment.Spam},
		assessedAddress{"https://clean.example/", spamassessment.Clean},
		assessedAddress{"https://second-spam.example/", spamassessment.Spam},
		assessedAddress{"https://unread.example/", spamassessment.Unassessed},
	))

	wantedAddresses := []string{
		"https://clean.example/",
		"https://unread.example/",
		"https://first-spam.example/",
		"https://second-spam.example/",
	}
	if addresses := addressesOf(orderedDocuments); !slices.Equal(addresses, wantedAddresses) {
		t.Fatalf("the ordering gives %v, want %v", addresses, wantedAddresses)
	}
}
