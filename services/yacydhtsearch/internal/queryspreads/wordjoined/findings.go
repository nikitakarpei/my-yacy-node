package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentsperword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func findingsFrom(
	query searchquery.Query,
	answered []documentasks.AnsweredWordPartition,
	documentsPerWord documentsperword.DocumentsPerWord,
	joinedDocuments yacymodel.URLHashes,
	urlMetadataAnswers urlmetadataasks.Answers,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:    query.WordHashes(),
		CompoundWords: query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(
			answered,
			joinedDocuments,
			urlMetadataAnswers,
		),
		DocumentsHeldPerQueryWord: documentsPerWord.AmountHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	answered []documentasks.AnsweredWordPartition,
	joinedDocuments yacymodel.URLHashes,
	urlMetadataAnswers urlmetadataasks.Answers,
) []queryfindings.FoundDocument {
	documentsThePeersSent := queryfindings.EmptyDocumentsThePeersSent()
	for _, answeredWordPartition := range answered {
		keepTheJoinedDocumentsListedWithMetadataIn(
			documentsThePeersSent, answeredWordPartition, joinedDocuments,
		)
	}
	for _, sentMetadata := range urlMetadataAnswers.MetadataThePeersSent() {
		documentsThePeersSent.KeepMetadataThePeerSent(sentMetadata.Metadata, sentMetadata.Peer)
	}

	return documentsThePeersSent.FoundDocuments()
}

func keepTheJoinedDocumentsListedWithMetadataIn(
	documentsThePeersSent *queryfindings.DocumentsThePeersSent,
	answeredWordPartition documentasks.AnsweredWordPartition,
	joinedDocuments yacymodel.URLHashes,
) {
	for _, answer := range answeredWordPartition.Answers {
		for _, listedDocument := range answer.ListedDocuments {
			metadata, sent := listedDocument.Metadata.Get()
			if !sent || !joinedDocuments.Contains(listedDocument.Hash) {
				continue
			}
			documentsThePeersSent.KeepDocumentThePeerListed(
				answer.Replica.Hash, answeredWordPartition.Word, metadata, listedDocument.Posting,
			)
		}
	}
}
