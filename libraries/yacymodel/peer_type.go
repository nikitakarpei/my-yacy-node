package yacymodel

import (
	"errors"
	"fmt"
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

func ParsePeerType(s string) (PeerType, error) {
	switch (PeerType{value: s}) {
	case PeerVirgin, PeerJunior, PeerMentee, PeerSenior, PeerMentor, PeerPrincipal:
		return PeerType{value: s}, nil
	default:
		return PeerType{}, fmt.Errorf("%w: %q", ErrInvalidPeerType, s)
	}
}

func (t PeerType) String() string { return t.value }
