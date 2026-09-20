package queryanswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type PostingReplica struct {
	Holder  yacymodel.Hash
	Word    yacymodel.Optional[yacymodel.Hash]
	Posting yacymodel.RWIPosting
}

type PostingReplicasPerDocument map[yacymodel.URLHash][]PostingReplica

func (p PostingReplicasPerDocument) Keep(
	document yacymodel.URLHash,
	replica PostingReplica,
) {
	p[document] = append(p[document], replica)
}

func (p PostingReplicasPerDocument) withoutDocuments(
	documents map[yacymodel.URLHash]struct{},
) PostingReplicasPerDocument {
	postingReplicasPerDocument := make(PostingReplicasPerDocument, len(p))
	for document, replicas := range p {
		if _, left := documents[document]; left {
			continue
		}
		postingReplicasPerDocument[document] = replicas
	}

	return postingReplicasPerDocument
}
