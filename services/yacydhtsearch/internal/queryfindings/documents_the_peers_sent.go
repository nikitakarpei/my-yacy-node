package queryfindings

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentsThePeersSent struct {
	documentsInFoundOrder       []yacymodel.URLHash
	metadataReplicasPerDocument map[yacymodel.URLHash][]MetadataReplica
	postingReplicasPerDocument  map[yacymodel.URLHash][]PostingReplica
}

func EmptyDocumentsThePeersSent() *DocumentsThePeersSent {
	return &DocumentsThePeersSent{
		metadataReplicasPerDocument: map[yacymodel.URLHash][]MetadataReplica{},
		postingReplicasPerDocument:  map[yacymodel.URLHash][]PostingReplica{},
	}
}

func (documents *DocumentsThePeersSent) KeepMetadataThePeerSent(
	metadata yacymodel.URLMetadata,
	peer yacymodel.Hash,
) {
	document := metadata.Hash
	if _, alreadyFound := documents.metadataReplicasPerDocument[document]; !alreadyFound {
		documents.documentsInFoundOrder = append(documents.documentsInFoundOrder, document)
	}
	documents.metadataReplicasPerDocument[document] = append(
		documents.metadataReplicasPerDocument[document],
		MetadataReplica{Holder: peer, Metadata: metadata},
	)
}

func (documents *DocumentsThePeersSent) KeepDocumentThePeerMatched(
	peer yacymodel.Hash,
	word yacymodel.Hash,
	metadata yacymodel.URLMetadata,
	posting yacymodel.Optional[yacymodel.RWIPosting],
) {
	documents.KeepMetadataThePeerSent(metadata, peer)
	sentPosting, sent := posting.Get()
	if !sent {
		return
	}
	documents.postingReplicasPerDocument[metadata.Hash] = append(
		documents.postingReplicasPerDocument[metadata.Hash],
		PostingReplica{Holder: peer, Word: word, Posting: sentPosting},
	)
}

func (documents *DocumentsThePeersSent) Among(
	chosenDocuments map[yacymodel.URLHash]struct{},
) *DocumentsThePeersSent {
	documentsAmongTheChosen := EmptyDocumentsThePeersSent()
	for _, document := range documents.documentsInFoundOrder {
		if _, chosen := chosenDocuments[document]; !chosen {
			continue
		}
		documentsAmongTheChosen.documentsInFoundOrder = append(
			documentsAmongTheChosen.documentsInFoundOrder, document,
		)
		documentsAmongTheChosen.metadataReplicasPerDocument[document] = slices.Clone(
			documents.metadataReplicasPerDocument[document],
		)
		documentsAmongTheChosen.postingReplicasPerDocument[document] = slices.Clone(
			documents.postingReplicasPerDocument[document],
		)
	}

	return documentsAmongTheChosen
}

func (documents *DocumentsThePeersSent) FoundDocuments() []FoundDocument {
	if len(documents.documentsInFoundOrder) == 0 {
		return nil
	}

	foundDocuments := make([]FoundDocument, 0, len(documents.documentsInFoundOrder))
	for _, document := range documents.documentsInFoundOrder {
		foundDocuments = append(foundDocuments, FoundDocumentOf(
			document,
			documents.metadataReplicasPerDocument[document],
			documents.postingReplicasPerDocument[document],
		))
	}

	return foundDocuments
}
