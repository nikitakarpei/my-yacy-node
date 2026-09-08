package peerannouncement

type AnnouncementObserver interface {
	ObserveAnnounceRound(amountOfPeersPerContactOutcome map[ContactOutcome]int)
}

type discardObserver struct{}

func (discardObserver) ObserveAnnounceRound(map[ContactOutcome]int) {}

var DiscardObserver AnnouncementObserver = discardObserver{}
