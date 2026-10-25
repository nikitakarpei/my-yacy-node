package peerannouncement

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"

type AnnouncementObserver interface {
	ObservePeerContacts(
		amountOfPeersPerContactOutcome map[ContactOutcome]int,
		amountOfReachablePeersPerScheme map[peerwire.Scheme]int,
	)
}

type discardObserver struct{}

func (discardObserver) ObservePeerContacts(map[ContactOutcome]int, map[peerwire.Scheme]int) {}

var DiscardObserver AnnouncementObserver = discardObserver{}
