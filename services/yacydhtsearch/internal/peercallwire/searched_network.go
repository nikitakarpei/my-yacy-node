package peercallwire

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type SearchedNetwork struct {
	Name           string
	RingPartitions yacymodel.DHTRingPartitions
}
