// Package peercontact decides which origins of a peer to try, in which order and
// within which time, for a greet, a back-ping probe and a delivery. A DNS name is
// tried over HTTPS first. It keeps in memory, per peer, the origin that answered
// a greet, and leads the other contacts with it.
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
	attempts := question.attemptsAt(originsFrom(addresses))
	answer, answered := c.exchange.Exchange(ctx, question.Request, attempts, greetBudget)
	if answered {
		c.answeredOrigins.record(question.Request.Peer, answer.Origin)
	} else {
		c.answeredOrigins.forget(question.Request.Peer)
	}

	return answer, answered
}

func (c *Contacts) Probe(
	ctx context.Context,
	addresses yacymodel.PeerAddresses,
	question Question,
) (peerwire.Answer, bool) {
	origins := originsFrom(addresses)
	if answeredOrigin, found := c.answeredOrigins.originAmong(
		question.Request.Peer,
		origins,
	); found {
		origins = append(
			[]peerwire.Origin{answeredOrigin},
			withoutOrigin(origins, answeredOrigin)...)
	}

	return c.exchange.Exchange(ctx, question.Request, question.attemptsAt(origins), probeBudget)
}

func (c *Contacts) Deliver(
	ctx context.Context,
	addresses yacymodel.PeerAddresses,
	question Question,
	acceptsFromAnsweredOrigin func(yacyproto.ResponseFields) bool,
) (peerwire.Answer, bool) {
	origins := originsFrom(addresses)
	attempts := question.attemptsAt(origins)
	if answeredOrigin, found := c.answeredOrigins.originAmong(
		question.Request.Peer,
		origins,
	); found {
		attempts = []peerwire.Attempt{{Origin: answeredOrigin, Accepts: acceptsFromAnsweredOrigin}}
	}

	return c.exchange.Exchange(ctx, question.Request, attempts, deliveryBudget)
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
