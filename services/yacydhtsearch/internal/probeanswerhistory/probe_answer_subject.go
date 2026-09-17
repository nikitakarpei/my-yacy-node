package probeanswerhistory

import "encoding/base64"

func SubjectOfEveryProbeAnswerIn(networkName string) string {
	return networkName + ".*.*"
}

func subjectOfProbeAnswerIn(networkName string, peerAtAddress PeerAtAddress) string {
	return networkName + "." + peerAtAddress.Hash.String() + "." +
		base64.RawURLEncoding.EncodeToString([]byte(peerAtAddress.Address))
}
