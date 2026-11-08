// Package natspermissions names what a service may publish and receive at a NATS broker that
// it does not own, built from the actions it takes there: pulling messages from a durable
// consumer, watching the backlog of a stream, publishing on subjects, and receiving on subjects.
// The broker enforces them as the permissions of the user that the service connects as.
package natspermissions

import "slices"

const (
	jetStreamAPIStart  = "$JS.API."
	streamInfoStart    = jetStreamAPIStart + "STREAM.INFO."
	consumerInfoStart  = jetStreamAPIStart + "CONSUMER.INFO."
	consumerListStart  = jetStreamAPIStart + "CONSUMER.LIST."
	nextMessageStart   = jetStreamAPIStart + "CONSUMER.MSG.NEXT."
	acknowledgeStart   = "$JS.ACK."
	everyTokenAfterEnd = ".>"
)

type Permissions struct {
	Publish   []string
	Subscribe []string
}

func PullingFrom(stream string, durableConsumer string) Permissions {
	consumer := stream + "." + durableConsumer
	return Permissions{Publish: []string{
		streamInfoStart + stream,
		consumerInfoStart + consumer,
		nextMessageStart + consumer,
		acknowledgeStart + consumer + everyTokenAfterEnd,
	}}
}

func WatchingBacklogOf(stream string) Permissions {
	return Permissions{Publish: []string{
		streamInfoStart + stream,
		consumerListStart + stream,
	}}
}

func PublishingOn(subjects ...string) Permissions {
	return Permissions{Publish: subjects}
}

func ReceivingOn(subjects ...string) Permissions {
	return Permissions{Subscribe: subjects}
}

func JoinedFrom(each ...Permissions) Permissions {
	var joined Permissions
	for _, permissions := range each {
		joined.Publish = appendedNew(joined.Publish, permissions.Publish)
		joined.Subscribe = appendedNew(joined.Subscribe, permissions.Subscribe)
	}
	return joined
}

func appendedNew(subjects []string, more []string) []string {
	for _, subject := range more {
		if !slices.Contains(subjects, subject) {
			subjects = append(subjects, subject)
		}
	}
	return subjects
}
