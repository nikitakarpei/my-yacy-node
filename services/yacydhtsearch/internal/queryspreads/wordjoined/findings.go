package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordholdings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func findingsFrom(
	query searchquery.Query,
	wordAnswers wordasks.Answers,
	holdings wordholdings.Holdings,
	joinedDocuments yacymodel.URLHashes,
	urlMetadata urlmetadataasks.Answers,
) queryfindings.Findings {
	return queryfindings.Findings{
		QueryWords:                query.WordHashes(),
		CompoundWords:             query.CompoundWords,
		FoundDocuments:            foundDocumentsFrom(wordAnswers, joinedDocuments, urlMetadata),
		DocumentsHeldPerQueryWord: holdings.AmountOfDocumentsHeldPerQueryWord(),
	}
}

func foundDocumentsFrom(
	wordAnswers wordasks.Answers,
	joinedDocuments yacymodel.URLHashes,
	urlMetadata urlmetadataasks.Answers,
) []queryfindings.FoundDocument {
	documentsThePeersSent := queryfindings.EmptyDocumentsThePeersSent()
	for _, matchedDocument := range wordAnswers.DocumentsThePeersMatched() {
		if !joinedDocuments.Contains(matchedDocument.Document) {
			continue
		}
		documentsThePeersSent.KeepDocumentThePeerMatched(
			matchedDocument.Replica,
			matchedDocument.Word,
			matchedDocument.Metadata,
			matchedDocument.Posting,
		)
	}
	for _, sentMetadata := range urlMetadata.MetadataThePeersSent() {
		documentsThePeersSent.KeepMetadataThePeerSent(sentMetadata.Metadata, sentMetadata.Peer)
	}

	return documentsThePeersSent.FoundDocuments()
}
