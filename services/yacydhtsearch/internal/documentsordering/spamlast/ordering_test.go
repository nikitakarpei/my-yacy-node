package spamlast_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/spamlast"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

type foundOrdering struct{}

func (foundOrdering) OrderedDocumentsOf(
	findings queryfindings.Findings,
) []queryfindings.FoundDocument {
	return slices.Clone(findings.FoundDocuments)
}

type assessedAddress struct {
	address string
	verdict pageclassification.Verdict
}

func findingsOf(assessedAddresses ...assessedAddress) queryfindings.Findings {
	findings := queryfindings.Findings{}
	for _, assessed := range assessedAddresses {
		findings.FoundDocuments = append(findings.FoundDocuments, queryfindings.FoundDocument{
			Address:     assessed.address,
			SpamVerdict: assessed.verdict,
		})
	}

	return findings
}

func addressesOf(foundDocuments []queryfindings.FoundDocument) []string {
	addresses := make([]string, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		addresses = append(addresses, foundDocument.Address)
	}

	return addresses
}

func TestSpamDocumentsComeAfterAllOthersInTheOrderOfTheOtherOrdering(t *testing.T) {
	t.Parallel()

	orderedDocuments := spamlast.New(foundOrdering{}).OrderedDocumentsOf(findingsOf(
		assessedAddress{"https://first-spam.example/", pageclassification.Positive},
		assessedAddress{"https://clean.example/", pageclassification.Negative},
		assessedAddress{"https://second-spam.example/", pageclassification.Positive},
		assessedAddress{"https://unread.example/", pageclassification.Unassessed},
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
