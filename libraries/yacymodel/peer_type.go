package yacymodel

import (
	"errors"
	"fmt"
	"slices"
)

type PeerType struct{ value string }

var (
	PeerVirgin    = PeerType{"virgin"}
	PeerJunior    = PeerType{"junior"}
	PeerMentee    = PeerType{"mentee"}
	PeerSenior    = PeerType{"senior"}
	PeerMentor    = PeerType{"mentor"}
	PeerPrincipal = PeerType{"principal"}
)

var ErrInvalidPeerType = errors.New("invalid peer type")

func PeerTypes() []PeerType {
	return []PeerType{
		PeerVirgin,
		PeerJunior,
		PeerMentee,
		PeerSenior,
		PeerMentor,
		PeerPrincipal,
	}
}

func ParsePeerType(s string) (PeerType, error) {
	peerType := PeerType{value: s}
	if !slices.Contains(PeerTypes(), peerType) {
		return PeerType{}, fmt.Errorf("%w: %q", ErrInvalidPeerType, s)
	}

	return peerType, nil
}

func (t PeerType) String() string { return t.value }
