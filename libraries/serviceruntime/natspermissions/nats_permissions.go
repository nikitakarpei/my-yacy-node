// Package natspermissions names what a service may publish and subscribe to at a NATS broker
// that it does not own, built from its actions: pulling from a durable consumer, watching the
// backlog of a stream, publishing and subscribing. Each JetStream call, a read included, is a
// request that the service publishes; the answer arrives on the inbox of the service, which these
// permissions leave to the caller.
package natspermissions

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

func SubscribingTo(subjects ...string) Permissions {
	return Permissions{Subscribe: subjects}
}

func JoinedFrom(each ...Permissions) Permissions {
	var joined Permissions
	for _, permissions := range each {
		joined.Publish = append(joined.Publish, permissions.Publish...)
		joined.Subscribe = append(joined.Subscribe, permissions.Subscribe...)
	}
	return joined
}
