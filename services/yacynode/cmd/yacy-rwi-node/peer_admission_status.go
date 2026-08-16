package main

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodestatus"
)

type peerAdmissionStatus struct {
	status      nodestatus.RuntimeStatus
	networkName string
}

func (s peerAdmissionStatus) NetworkName(context.Context) string {
	return s.networkName
}

func (s peerAdmissionStatus) SelfSeed(ctx context.Context) yacymodel.Seed {
	return s.status.SelfSeed(ctx)
}
