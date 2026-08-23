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

func derivation(
	sourceFormat, targetFormat documentextraction.Format,
) derivationreach.DerivationFormats {
	return derivationreach.DerivationFormats{
		SourceFormat: sourceFormat,
		TargetFormat: targetFormat,
	}
}

func TestEnsureNoCycleAcceptsDerivationsThatOnlyMoveForward(t *testing.T) {
	err := derivationreach.EnsureNoCycle([]derivationreach.DerivationFormats{
		derivation(documentHTML, readableHTML),
		derivation(readableHTML, markdown),
		derivation(documentHTML, markdown),
	})
	if err != nil {
		t.Fatalf("derivations that only move forward hold no cycle: %v", err)
	}
}

func TestEnsureNoCycleRejectsAFormatThatDerivesFromItself(t *testing.T) {
	err := derivationreach.EnsureNoCycle([]derivationreach.DerivationFormats{
		derivation(documentHTML, readableHTML),
		derivation(readableHTML, markdown),
		derivation(markdown, readableHTML),
	})
	if err == nil {
		t.Fatal("readable-html derives from markdown, which derives from readable-html")
	}
}

func TestEnsureNoDanglingFormatAcceptsAFullyConnectedCatalog(t *testing.T) {
	err := derivationreach.EnsureNoDanglingFormat(
		[]derivationreach.DerivationFormats{
			derivation(documentHTML, readableHTML),
			derivation(readableHTML, markdown),
		},
		[]documentextraction.Format{documentHTML},
	)
	if err != nil {
		t.Fatalf("document-html reaches every target format: %v", err)
	}
}

func TestEnsureNoDanglingFormatRejectsATargetNoEmittedFormatReaches(t *testing.T) {
	err := derivationreach.EnsureNoDanglingFormat(
		[]derivationreach.DerivationFormats{
			derivation(documentHTML, readableHTML),
			derivation(transcript, markdown),
		},
		[]documentextraction.Format{documentHTML},
	)
	if err == nil {
		t.Fatal("markdown is only derivable from a format no extractor emits")
	}
}

func TestEnsureNoDanglingFormatRejectsAnEmittedFormatThatReachesNoTarget(t *testing.T) {
	err := derivationreach.EnsureNoDanglingFormat(
		[]derivationreach.DerivationFormats{
			derivation(documentHTML, readableHTML),
		},
		[]documentextraction.Format{documentHTML, transcript},
	)
	if err == nil {
		t.Fatal("audio-transcript derives no target format")
	}
}

func TestEnsureNoDanglingFormatReachesATargetThroughAChain(t *testing.T) {
	err := derivationreach.EnsureNoDanglingFormat(
		[]derivationreach.DerivationFormats{
			derivation(documentHTML, readableHTML),
			derivation(readableHTML, markdown),
		},
		[]documentextraction.Format{documentHTML},
	)
	if err != nil {
		t.Fatalf("markdown is reachable through readable-html: %v", err)
	}
}
