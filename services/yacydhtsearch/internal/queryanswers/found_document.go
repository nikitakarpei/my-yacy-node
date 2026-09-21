package queryanswers

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type FoundDocument struct {
	Hash             yacymodel.URLHash
	Address          string
	Title            string
	Snippet          string
	PublishedAt      yacymodel.Optional[time.Time]
	FaviconAddress   string
	MetadataReplicas []MetadataReplica
	PostingReplicas  []PostingReplica
	Facts            DocumentFacts
}

func FoundDocumentOf(
	document yacymodel.URLHash,
	metadataReplicas []MetadataReplica,
	postingReplicas []PostingReplica,
) FoundDocument {
	shownMetadata := metadataShownAmong(metadataReplicas)

	return FoundDocument{
		Hash:             document,
		Address:          shownMetadata.Address,
		Title:            shownMetadata.Title,
		Snippet:          shownMetadata.Snippet,
		PublishedAt:      publicationInstantOf(shownMetadata),
		FaviconAddress:   shownMetadata.FaviconAddress,
		MetadataReplicas: metadataReplicas,
		PostingReplicas:  postingReplicas,
		Facts:            documentFactsOfFirstReplicaOfEachWord(postingReplicas),
	}
}

func metadataShownAmong(metadataReplicas []MetadataReplica) yacymodel.URLMetadata {
	if len(metadataReplicas) == 0 {
		return yacymodel.URLMetadata{}
	}

	return metadataReplicas[0].Metadata
}

func publicationInstantOf(metadata yacymodel.URLMetadata) yacymodel.Optional[time.Time] {
	day, ok := metadata.Modified.Get()
	if !ok {
		day, ok = metadata.Loaded.Get()
	}
	if !ok {
		return yacymodel.None[time.Time]()
	}

	return yacymodel.Some(day.Time())
}

func (f FoundDocument) withItsReadPage(pageContents pagecontents.PageContents) FoundDocument {
	if pageContents.Address != "" {
		f.Address = pageContents.Address
	}
	if pageContents.Title != "" {
		f.Title = pageContents.Title
	}
	f.Snippet = pageContents.Snippet
	f.Facts = documentFactsOfReadPage(pageContents)

	return f
}
