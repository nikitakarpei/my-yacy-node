// Package applog reports how a peer stands on a form of an ask, and how its
// answer was judged, to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

const (
	msgPeerStood  = "peer stands on a form of an ask"
	msgPeerJudged = "peer judged on a form of an ask"
)

type JudgementsLog struct{}

func (JudgementsLog) PeerStood(
	ctx context.Context,
	form peerjudgements.Form,
	standing peerjudgements.PeerStanding,
) {
	slog.DebugContext(ctx, msgPeerStood,
		slog.String("form", string(form)),
		slog.String("peer", standing.Peer.String()),
		slog.String("versionClaimed", standing.Version),
		slog.String("standing", string(standing.Standing)),
	)
}

func (JudgementsLog) PeerJudged(
	ctx context.Context,
	form peerjudgements.Form,
	judgedPeer peerjudgements.JudgedPeer,
) {
	slog.DebugContext(ctx, msgPeerJudged,
		slog.String("form", string(form)),
		slog.String("peer", judgedPeer.Peer.String()),
		slog.String("versionClaimed", judgedPeer.Version),
		slog.String("judgement", string(judgedPeer.Judgement)),
	)
}
