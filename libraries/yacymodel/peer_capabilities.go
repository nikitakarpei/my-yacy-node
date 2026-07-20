package yacymodel

type PeerCapabilities struct {
	DirectConnect     bool
	AcceptRemoteCrawl bool
	AcceptRemoteIndex bool
	RootNode          bool
	SSLAvailable      bool
}
