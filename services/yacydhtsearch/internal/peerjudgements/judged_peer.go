package peerjudgements

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type Judgement string

const (
	Honored    Judgement = "honored"
	Ignored    Judgement = "ignored"
	NoEvidence Judgement = "no evidence"
)

type JudgedPeer struct {
	PeerAtVersion
	Judgement Judgement
}

func JudgedPeerFrom(peer yacymodel.Hash, version string, judgement Judgement) JudgedPeer {
	return JudgedPeer{
		PeerAtVersion: PeerAtVersion{Peer: peer, Version: version},
		Judgement:     judgement,
	}
}

func NoEvidenceFrom(peer yacymodel.Hash) JudgedPeer {
	return JudgedPeer{PeerAtVersion: PeerAtVersion{Peer: peer}, Judgement: NoEvidence}
}
