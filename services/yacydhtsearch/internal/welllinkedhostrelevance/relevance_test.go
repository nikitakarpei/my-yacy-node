package welllinkedhostrelevance_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/welllinkedhostrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type relevancePerAddress map[string]float64

func (relevance relevancePerAddress) RelevancePerDocumentOf(
	_ context.Context,
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	relevancePerDocument := map[yacymodel.URLHash]float64{}
	for _, document := range answers.FoundDocuments {
		relevancePerDocument[document.Hash] = relevance[document.Address]
	}

	return relevancePerDocument
}

type listedAddresses map[string]bool

func (listed listedAddresses) HoldsHostOf(address string) bool {
	return listed[address]
}

type demotedDocumentsRecord struct {
	amountsOfDemotedDocuments []int
}

func (record *demotedDocumentsRecord) DocumentsDemoted(
	_ context.Context,
	amountOfDemotedDocuments int,
) {
	record.amountsOfDemotedDocuments = append(
		record.amountsOfDemotedDocuments, amountOfDemotedDocuments,
	)
}

func answersFoundAt(t *testing.T, addresses ...string) queryanswers.AnsweredQuery {
	t.Helper()

	answers := queryanswers.AnsweredQuery{}
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			t.Fatalf("hash %q: %v", address, err)
		}
		answers.FoundDocuments = append(
			answers.FoundDocuments,
			queryanswers.FoundDocument{Hash: hash, Address: address},
		)
	}

	return answers
}

func TestADocumentOfAWellLinkedHostKeepsItsRelevance(t *testing.T) {
	t.Parallel()

	const address = "https://kernel.org/"
	answers := answersFoundAt(t, address)
	relevance := welllinkedhostrelevance.New(
		relevancePerAddress{address: 2},
		listedAddresses{address: true},
		&demotedDocumentsRecord{},
	)

	got := relevance.RelevancePerDocumentOf(t.Context(), answers)[answers.FoundDocuments[0].Hash]
	if got != 2 {
		t.Fatalf("relevance = %v, want the 2 it had", got)
	}
}

func TestADocumentOfAnotherHostLosesATenthOfItsRelevance(t *testing.T) {
	t.Parallel()

	const address = "https://obscure.example/"
	answers := answersFoundAt(t, address)
	relevance := welllinkedhostrelevance.New(
		relevancePerAddress{address: 2},
		listedAddresses{},
		&demotedDocumentsRecord{},
	)

	got := relevance.RelevancePerDocumentOf(t.Context(), answers)[answers.FoundDocuments[0].Hash]
	if got != 1.8 {
		t.Fatalf("relevance = %v, want 1.8", got)
	}
}

func TestARelevanceOfZeroOrLessStaysAsItIs(t *testing.T) {
	t.Parallel()

	const nothing, negative = "https://nothing.example/", "https://negative.example/"
	answers := answersFoundAt(t, nothing, negative)
	relevance := welllinkedhostrelevance.New(
		relevancePerAddress{nothing: 0, negative: -1},
		listedAddresses{},
		&demotedDocumentsRecord{},
	)

	relevancePerDocument := relevance.RelevancePerDocumentOf(t.Context(), answers)
	if relevancePerDocument[answers.FoundDocuments[0].Hash] != 0 ||
		relevancePerDocument[answers.FoundDocuments[1].Hash] != -1 {
		t.Fatalf("relevance = %v, want 0 and -1 as they were", relevancePerDocument)
	}
}

func TestTheObserverLearnsHowManyDocumentsOfAQueryWereDemoted(t *testing.T) {
	t.Parallel()

	const listed, first, second, irrelevant = "https://listed.example/",
		"https://first.example/", "https://second.example/", "https://irrelevant.example/"
	record := &demotedDocumentsRecord{}
	relevance := welllinkedhostrelevance.New(
		relevancePerAddress{listed: 1, first: 1, second: 1, irrelevant: 0},
		listedAddresses{listed: true},
		welllinkedhostrelevance.RelevanceObservers{record},
	)

	relevance.RelevancePerDocumentOf(
		t.Context(),
		answersFoundAt(t, listed, first, second, irrelevant),
	)

	if len(record.amountsOfDemotedDocuments) != 1 || record.amountsOfDemotedDocuments[0] != 2 {
		t.Fatalf("demoted amounts = %v, want one query with 2", record.amountsOfDemotedDocuments)
	}
}
