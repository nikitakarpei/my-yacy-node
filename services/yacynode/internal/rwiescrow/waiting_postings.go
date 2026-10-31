package rwiescrow

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

type WaitingPostings struct {
	Proven   []WaitingProvenPosting
	Unproven []WaitingPosting
}

func (w *WaitingPostings) add(
	ctx context.Context,
	identity postingIdentity,
	escrowed escrowedPosting,
) {
	posting := escrowed.Posting
	posting.WordHash = identity.Word
	posting.URLHash = identity.URL
	escrowedProof, proven := escrowed.Proof.Get()
	if !proven {
		w.Unproven = append(w.Unproven, WaitingPosting{
			origin:   escrowed.Origin,
			posting:  posting,
			identity: identity,
		})

		return
	}
	provenPosting, err := yacymodel.ProvenPostingFrom(posting, escrowedProof)
	if err != nil {
		reportRefusedProof(ctx, posting, err)

		return
	}
	w.Proven = append(w.Proven, WaitingProvenPosting{
		origin:        escrowed.Origin,
		provenPosting: provenPosting,
		identity:      identity,
	})
}

func reportRefusedProof(ctx context.Context, posting yacymodel.RWIPosting, err error) {
	slog.WarnContext(ctx, "escrowed proof refused, posting left to expire",
		slog.String("url", posting.URLHash.String()),
		slog.String("word", posting.WordHash.String()),
		slog.Any("error", err),
	)
}

type WaitingProvenPosting struct {
	origin        rwiorigin.Origin
	provenPosting yacymodel.ProvenPosting
	identity      postingIdentity
}

func (w WaitingProvenPosting) Origin() rwiorigin.Origin {
	return w.origin
}

func (w WaitingProvenPosting) ProvenPosting() yacymodel.ProvenPosting {
	return w.provenPosting
}

type WaitingPosting struct {
	origin   rwiorigin.Origin
	posting  yacymodel.RWIPosting
	identity postingIdentity
}

func (w WaitingPosting) Origin() rwiorigin.Origin {
	return w.origin
}

func (w WaitingPosting) Posting() yacymodel.RWIPosting {
	return w.posting
}
