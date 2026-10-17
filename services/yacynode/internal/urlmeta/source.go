package urlmeta

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"

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
