package main

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
)

// TECHDEBT: naming — the file name says adapters, an umbrella word; it declares only peeringStatus
type peeringStatus struct {
	status      nodestatus.RuntimeStatus
	networkName string
}

func (s peeringStatus) NetworkName(context.Context) string {
	return s.networkName
}

func (s peeringStatus) SelfSeed(ctx context.Context) yacymodel.Seed {
	return s.status.SelfSeed(ctx)
}
