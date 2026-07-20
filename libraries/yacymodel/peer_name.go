package yacymodel

import (
	"errors"
	"fmt"
)

const (
	peerNameMinLength = 3
	peerNameMaxLength = 80
)

var ErrBadPeerName = errors.New("bad peer name")

type PeerName struct {
	value string
}

func ParsePeerName(s string) (PeerName, error) {
	if len(s) < peerNameMinLength || len(s) > peerNameMaxLength {
		return PeerName{}, fmt.Errorf(
			"%w: length %d, want %d-%d",
			ErrBadPeerName,
			len(s),
			peerNameMinLength,
			peerNameMaxLength,
		)
	}
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' ||
			r == '_' {
			continue
		}
		return PeerName{}, fmt.Errorf("%w: %q", ErrBadPeerName, s)
	}
	return PeerName{value: s}, nil
}

func (n PeerName) String() string { return n.value }
