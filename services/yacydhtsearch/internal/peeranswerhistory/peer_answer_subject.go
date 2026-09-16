package peeranswerhistory

import "encoding/base64"

func SubjectOfEveryPeerAnswerIn(networkName string) string {
	return networkName + ".*.*"
}

func subjectOfPeerAnswerIn(networkName string, peerAtAddress PeerAtAddress) string {
	return networkName + "." + peerAtAddress.Hash.String() + "." +
		base64.RawURLEncoding.EncodeToString([]byte(peerAtAddress.Address))
}
