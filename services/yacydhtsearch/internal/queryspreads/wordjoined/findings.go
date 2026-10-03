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
	documentAnswers documentasks.Answers,
	documentsPerWord documentsperword.DocumentsPerWord,
	joinedDocuments yacymodel.URLHashes,
	urlMetadataAnswers urlmetadataasks.Answers,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:    query.WordHashes(),
		CompoundWords: query.CompoundWords,
		FoundDocuments: foundDocumentsFrom(
			documentAnswers,
			joinedDocuments,
			urlMetadataAnswers,
		),
		DocumentsHeldPerQueryWord: documentsPerWord.AmountHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	documentAnswers documentasks.Answers,
	joinedDocuments yacymodel.URLHashes,
	urlMetadataAnswers urlmetadataasks.Answers,
) []queryfindings.FoundDocument {
	documentsThePeersSent := queryfindings.EmptyDocumentsThePeersSent()
	for _, listedDocumentWithMetadata := range documentAnswers.ListedDocumentsWithMetadata() {
		if !joinedDocuments.Contains(listedDocumentWithMetadata.Document) {
			continue
		}
		documentsThePeersSent.KeepDocumentThePeerListed(
			listedDocumentWithMetadata.Replica,
			listedDocumentWithMetadata.Word,
			listedDocumentWithMetadata.Metadata,
			listedDocumentWithMetadata.Posting,
		)
	}
	for _, sentMetadata := range urlMetadataAnswers.MetadataThePeersSent() {
		documentsThePeersSent.KeepMetadataThePeerSent(sentMetadata.Metadata, sentMetadata.Peer)
	}

	return documentsThePeersSent.FoundDocuments()
}
