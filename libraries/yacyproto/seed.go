package yacyproto

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// EncodeSeed renders a seed as the base64-framed property-form line YaCy peers
// exchange.
func EncodeSeed(seed yacymodel.Seed) string {
	return seedWireCodec{}.encode(seed)
}

// ParseSeed reads a base64-framed seed line into the domain model.
func ParseSeed(ctx context.Context, framed string) (yacymodel.Seed, error) {
	return seedWireCodec{}.decode(ctx, framed)
}

// ParseSoftwareVersion reads YaCy's packed version double into the domain model.
func ParseSoftwareVersion(text string) (yacymodel.SoftwareVersion, error) {
	return softwareVersionWireCodec{}.decode(text)
}

// ParseRemoteSeed reads a seed line and additionally requires the peer to be
// reachable, the way YaCy rejects a remote seed that carries no address.
func ParseRemoteSeed(ctx context.Context, framed string) (yacymodel.Seed, error) {
	return seedWireCodec{}.decodeRemote(ctx, framed)
}
