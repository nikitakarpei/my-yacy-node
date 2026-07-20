package yacyproto

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

const (
	peerFlagsWidth       = 4
	peerFlagsBitsPerChar = 5
	peerFlagsCharOffset  = 32

	peerFlagDirectConnect     = 0
	peerFlagAcceptRemoteCrawl = 1
	peerFlagAcceptRemoteIndex = 2
	peerFlagRootNode          = 3
	peerFlagSSLAvailable      = 4
)

// peerCapabilitiesWireCodec translates between the peer capabilities domain type
// and YaCy's four-character Flags field, each character carrying five bits above
// the printable-ASCII offset.
type peerCapabilitiesWireCodec struct{}

func (peerCapabilitiesWireCodec) decode(text string) yacymodel.PeerCapabilities {
	bit := func(index int) bool {
		char := index / peerFlagsBitsPerChar
		if char >= len(text) {
			return false
		}
		value := int(text[char]) - peerFlagsCharOffset
		return value&(1<<(index%peerFlagsBitsPerChar)) != 0
	}
	return yacymodel.PeerCapabilities{
		DirectConnect:     bit(peerFlagDirectConnect),
		AcceptRemoteCrawl: bit(peerFlagAcceptRemoteCrawl),
		AcceptRemoteIndex: bit(peerFlagAcceptRemoteIndex),
		RootNode:          bit(peerFlagRootNode),
		SSLAvailable:      bit(peerFlagSSLAvailable),
	}
}

func (peerCapabilitiesWireCodec) encode(c yacymodel.PeerCapabilities) string {
	chars := make([]byte, peerFlagsWidth)
	for i := range chars {
		chars[i] = peerFlagsCharOffset
	}
	set := func(index int, value bool) {
		if !value {
			return
		}
		chars[index/peerFlagsBitsPerChar] |= 1 << (index % peerFlagsBitsPerChar)
	}
	set(peerFlagDirectConnect, c.DirectConnect)
	set(peerFlagAcceptRemoteCrawl, c.AcceptRemoteCrawl)
	set(peerFlagAcceptRemoteIndex, c.AcceptRemoteIndex)
	set(peerFlagRootNode, c.RootNode)
	set(peerFlagSSLAvailable, c.SSLAvailable)
	return string(chars)
}
