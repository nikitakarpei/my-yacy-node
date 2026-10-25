package yacymodel

type PeerAddresses struct {
	Hosts      []Host
	Port       Port
	SecurePort Optional[Port]
}
