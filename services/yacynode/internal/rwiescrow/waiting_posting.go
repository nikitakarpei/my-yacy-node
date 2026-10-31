package rwiescrow

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
)

type WaitingPosting struct {
	pagePosting yacymodel.PagePosting
	version     pageversion.Version
	identity    postingIdentity
}

func waitingPostingFrom(
	identity postingIdentity,
	escrowed escrowedPosting,
) (WaitingPosting, error) {
	posting := escrowed.Posting
	posting.WordHash = identity.Word
	posting.URLHash = identity.URL
	pagePosting, err := pagePostingFrom(posting, escrowed.Proof)
	if err != nil {
		return WaitingPosting{}, err
	}

	return WaitingPosting{
		pagePosting: pagePosting,
		version: pageversion.FromPagePosting(
			pagePosting,
			pageversion.SourcePeer(escrowed.Origin),
		),
		identity: identity,
	}, nil
}

func pagePostingFrom(
	posting yacymodel.RWIPosting,
	proof yacymodel.Optional[yacymodel.PostingsProof],
) (yacymodel.PagePosting, error) {
	escrowedProof, proven := proof.Get()
	if !proven {
		return yacymodel.UnprovenPagePostingFrom(posting)
	}

	return yacymodel.ProvenPagePostingFrom(posting, escrowedProof)
}

func (w WaitingPosting) PagePosting() yacymodel.PagePosting {
	return w.pagePosting
}

func (w WaitingPosting) Version() pageversion.Version {
	return w.version
}
