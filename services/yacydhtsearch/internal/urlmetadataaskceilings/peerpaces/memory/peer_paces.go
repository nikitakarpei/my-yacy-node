// Package memory remembers in this process the pace of each peer in its URL
// metadata answers.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerPaces struct {
	mutex             sync.Mutex
	paceOfEachAddress map[string]time.Duration
}

func New() *PeerPaces {
	return &PeerPaces{paceOfEachAddress: map[string]time.Duration{}}
}

func (paces *PeerPaces) Read(
	_ context.Context,
	address string,
) yacymodel.Optional[time.Duration] {
	paces.mutex.Lock()
	defer paces.mutex.Unlock()

	return paces.paceOf(address)
}

func (paces *PeerPaces) paceOf(
	address string,
) yacymodel.Optional[time.Duration] {
	pace, remembered := paces.paceOfEachAddress[address]
	if !remembered {
		return yacymodel.None[time.Duration]()
	}

	return yacymodel.Some(pace)
}

func (paces *PeerPaces) Update(
	_ context.Context,
	address string,
	updated func(pace yacymodel.Optional[time.Duration]) time.Duration,
) time.Duration {
	paces.mutex.Lock()
	defer paces.mutex.Unlock()

	pace := updated(paces.paceOf(address))
	paces.paceOfEachAddress[address] = pace

	return pace
}
