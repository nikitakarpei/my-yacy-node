package derivationreach_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/derivationreach"
)

const (
	documentHTML = documentextraction.Format("document-html")
	readableHTML = documentextraction.Format("readable-html")
	markdown     = documentextraction.Format("markdown")
	transcript   = documentextraction.Format("audio-transcript")
)

type formatDerivation struct {
	sourceFormat documentextraction.Format
	targetFormat documentextraction.Format
}

func derivation(
	sourceFormat, targetFormat documentextraction.Format,
) formatDerivation {
	return formatDerivation{sourceFormat: sourceFormat, targetFormat: targetFormat}
}

func (d formatDerivation) SourceFormat() documentextraction.Format {
	return d.sourceFormat
}

func (d formatDerivation) TargetFormat() documentextraction.Format {
	return d.targetFormat
}

func TestEnsureFormatsDerivableAcceptsDerivationsThatOnlyMoveForward(t *testing.T) {
	err := derivationreach.EnsureFormatsDerivable(
		[]formatDerivation{
			derivation(documentHTML, readableHTML),
			derivation(readableHTML, markdown),
			derivation(documentHTML, markdown),
		},
		[]documentextraction.Format{documentHTML},
	)
	if err != nil {
		t.Fatalf("derivations that only move forward hold no cycle: %v", err)
	}
}

func TestEnsureFormatsDerivableRejectsAFormatThatDerivesFromItself(t *testing.T) {
	err := derivationreach.EnsureFormatsDerivable(
		[]formatDerivation{
			derivation(documentHTML, readableHTML),
			derivation(readableHTML, markdown),
			derivation(markdown, readableHTML),
		},
		[]documentextraction.Format{documentHTML},
	)
	if err == nil {
		t.Fatal("readable-html derives from markdown, which derives from readable-html")
	}
}

func TestEnsureFormatsDerivableReachesATargetThroughAChain(t *testing.T) {
	err := derivationreach.EnsureFormatsDerivable(
		[]formatDerivation{
			derivation(documentHTML, readableHTML),
			derivation(readableHTML, markdown),
		},
		[]documentextraction.Format{documentHTML},
	)
	if err != nil {
		t.Fatalf("markdown is reachable through readable-html: %v", err)
	}
}

func TestEnsureFormatsDerivableRejectsATargetNoEmittedFormatReaches(t *testing.T) {
	err := derivationreach.EnsureFormatsDerivable(
		[]formatDerivation{
			derivation(documentHTML, readableHTML),
			derivation(transcript, markdown),
		},
		[]documentextraction.Format{documentHTML},
	)
	if err == nil {
		t.Fatal("markdown is only derivable from a format no extractor emits")
	}
}

func TestEnsureFormatsDerivableAcceptsAnEmittedFormatThatDerivesNoTarget(t *testing.T) {
	err := derivationreach.EnsureFormatsDerivable(
		[]formatDerivation{
			derivation(documentHTML, readableHTML),
		},
		[]documentextraction.Format{documentHTML, markdown},
	)
	if err != nil {
		t.Fatalf("an extractor may emit a format that needs no further derivation: %v", err)
	}
}
