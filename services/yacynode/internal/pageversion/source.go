package pageversion

import (
	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

type sourceKind byte

const (
	sourceUnknown sourceKind = iota
	sourceThisNode
	sourcePeer
)

type Source struct {
	kind sourceKind
	peer rwiorigin.Origin
}

func SourceThisNode() Source {
	return Source{kind: sourceThisNode}
}

func SourcePeer(origin rwiorigin.Origin) Source {
	return Source{kind: sourcePeer, peer: origin}
}

func WriteSource(stored *storedfields.Writer, source Source) {
	stored.Byte(byte(source.kind))
	if source.kind == sourcePeer {
		rwiorigin.Write(stored, source.peer)
	}
}

func SourceFrom(stored *storedfields.Reader) Source {
	source := Source{kind: sourceKind(stored.Byte("source"))}
	if source.kind == sourcePeer {
		source.peer = rwiorigin.From(stored)
	}

	return source
}
