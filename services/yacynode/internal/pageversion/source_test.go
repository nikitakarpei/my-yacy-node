package pageversion_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

var errBadRecord = errors.New("bad record")

func sourceFrom(t *testing.T, record []byte) pageversion.Source {
	t.Helper()

	stored := storedfields.ReaderOf(record, errBadRecord)
	source := pageversion.SourceFrom(stored)
	if err := stored.Err(); err != nil {
		t.Fatalf("SourceFrom: %v", err)
	}

	return source
}

func TestStoredSourceReadsBackAsWritten(t *testing.T) {
	for name, source := range map[string]pageversion.Source{
		"this node": pageversion.SourceThisNode(),
		"peer":      pageversion.SourcePeer(sendingNetwork),
	} {
		t.Run(name, func(t *testing.T) {
			var stored storedfields.Writer
			pageversion.WriteSource(&stored, source)

			if read := sourceFrom(t, stored.Record()); read != source {
				t.Fatalf("read back %+v, want %+v", read, source)
			}
		})
	}
}

func TestSourceStoredByEarlierNodesReadsAsTheSameSource(t *testing.T) {
	var thisNode storedfields.Writer
	thisNode.Byte(1)
	var peer storedfields.Writer
	peer.Byte(2)
	rwiorigin.Write(&peer, sendingNetwork)

	if sourceFrom(t, thisNode.Record()) != pageversion.SourceThisNode() ||
		sourceFrom(t, peer.Record()) != pageversion.SourcePeer(sendingNetwork) {
		t.Fatal("a source stored by an earlier node reads as another source")
	}
}
