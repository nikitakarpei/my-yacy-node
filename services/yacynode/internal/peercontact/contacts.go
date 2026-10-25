// Package peercontact decides which origins of a peer to try, in which order and
// within which time. A greet and a back-ping probe try the origins in turn, a DNS
// name over HTTPS first, and keep in memory, per peer, the origin that answered.
// A delivery goes only to that origin.
package peercontact

import (
	"context"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	greetBudget         = 30 * time.Second
	probeBudget         = 13 * time.Second
	deliveryBudget      = 30 * time.Second
	attemptsPerQuestion = 5
	hostsPerPeer        = 4
	httpsPort           = yacymodel.Port(443)
)

type Question struct {
	Request peerwire.Request
	Accepts func(yacyproto.ResponseFields) bool
}

type Contacts struct {
	exchange        peerwire.FormExchange
	answeredOrigins *answeredOrigins
}

func New(exchange peerwire.FormExchange, capacity int) *Contacts {
	return &Contacts{exchange: exchange, answeredOrigins: newAnsweredOrigins(capacity)}
}

func (c *Contacts) Greet(
	ctx context.Context,
	addresses yacymodel.PeerAddresses,
	question Question,
) (peerwire.Answer, bool) {
	return c.walk(ctx, question, originsFrom(addresses), greetBudget)
}

func (c *Contacts) Probe(
	ctx context.Context,
	addresses yacymodel.PeerAddresses,
	question Question,
) (peerwire.Answer, bool) {
	origins := originsFrom(addresses)
	if answeredOrigin, found := c.answeredOrigins.originOf(question.Request.Peer); found &&
		slices.Contains(origins, answeredOrigin) {
		origins = append(
			[]peerwire.Origin{answeredOrigin},
			withoutOrigin(origins, answeredOrigin)...)
	}

	return c.walk(ctx, question, origins, probeBudget)
}

func (c *Contacts) AnsweredOriginOf(peer yacymodel.Hash) (peerwire.Origin, bool) {
	return c.answeredOrigins.originOf(peer)
}

func (c *Contacts) Deliver(
	ctx context.Context,
	origin peerwire.Origin,
	question Question,
) (peerwire.Answer, bool) {
	return c.exchange.Exchange(
		ctx,
		question.Request,
		[]peerwire.Attempt{{Origin: origin, Accepts: question.Accepts}},
		deliveryBudget,
	)
}

func (c *Contacts) walk(
	ctx context.Context,
	question Question,
	origins []peerwire.Origin,
	budget time.Duration,
) (peerwire.Answer, bool) {
	answer, answered := c.exchange.Exchange(
		ctx,
		question.Request,
		question.attemptsAt(origins),
		budget,
	)
	if answered {
		c.answeredOrigins.record(question.Request.Peer, answer.Origin)
	} else {
		c.answeredOrigins.forget(question.Request.Peer)
	}

	return answer, answered
}

func originsFrom(addresses yacymodel.PeerAddresses) []peerwire.Origin {
	var origins []peerwire.Origin
	for _, host := range addresses.Hosts[:min(len(addresses.Hosts), hostsPerPeer)] {
		if _, isAddr := host.Addr(); !isAddr {
			origins = append(
				origins,
				peerwire.Origin{Scheme: peerwire.HTTPS, Host: host, Port: httpsPort},
			)
			if securePort, announced := addresses.SecurePort.Get(); announced &&
				securePort != httpsPort {
				origins = append(
					origins,
					peerwire.Origin{Scheme: peerwire.HTTPS, Host: host, Port: securePort},
				)
			}
		}
		origins = append(
			origins,
			peerwire.Origin{Scheme: peerwire.HTTP, Host: host, Port: addresses.Port},
		)
	}

	return origins
}

func (q Question) attemptsAt(origins []peerwire.Origin) []peerwire.Attempt {
	attempts := make([]peerwire.Attempt, 0, attemptsPerQuestion)
	for _, origin := range origins[:min(len(origins), attemptsPerQuestion)] {
		attempts = append(attempts, peerwire.Attempt{Origin: origin, Accepts: q.Accepts})
	}

	return attempts
}

func withoutOrigin(origins []peerwire.Origin, excluded peerwire.Origin) []peerwire.Origin {
	return slices.DeleteFunc(slices.Clone(origins), func(origin peerwire.Origin) bool {
		return origin == excluded
	})
}
