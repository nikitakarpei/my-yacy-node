package peereligibility

import "time"

const weightOfNewSendTime = 0.2

type peerLoad struct {
	amountOfSendsInFlight int
	usualSendTime         time.Duration
}

func (l peerLoad) withSendStarted() peerLoad {
	l.amountOfSendsInFlight++

	return l
}

func (l peerLoad) isAtLimitWithin(workInFlight time.Duration) bool {
	return l.usualSendTime > 0 &&
		l.amountOfSendsInFlight >= l.amountOfSendsAllowedWithin(workInFlight)
}

func (l peerLoad) amountOfSendsAllowedWithin(workInFlight time.Duration) int {
	return max(1, int(workInFlight/l.usualSendTime))
}

func (l peerLoad) withSendEndedAfter(sendTime time.Duration) peerLoad {
	l.amountOfSendsInFlight--
	l.usualSendTime = l.usualSendTimeWith(sendTime)

	return l
}

func (l peerLoad) usualSendTimeWith(sendTime time.Duration) time.Duration {
	if l.usualSendTime == 0 {
		return sendTime
	}

	return l.usualSendTime + time.Duration(weightOfNewSendTime*float64(sendTime-l.usualSendTime))
}
