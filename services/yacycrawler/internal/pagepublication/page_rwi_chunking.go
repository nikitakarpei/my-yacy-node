package pagepublication

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const postingsPerChunkLimit = 1000

func chunkPageRWI(
	representation yacycrawlcontract.PageRWIRepresentation,
) []yacycrawlcontract.PageRWIChunk {
	var chunks []yacycrawlcontract.PageRWIChunk
	if len(representation.Metadata) > 0 {
		chunks = append(chunks, yacycrawlcontract.PageRWIMetadataChunk{
			CanonicalURL: representation.CanonicalURL,
			Metadata:     representation.Metadata,
		})
	}
	for start := 0; start < len(representation.Postings); start += postingsPerChunkLimit {
		end := min(start+postingsPerChunkLimit, len(representation.Postings))
		chunks = append(chunks, yacycrawlcontract.PageRWIPostingChunk{
			CanonicalURL: representation.CanonicalURL,
			Postings:     representation.Postings[start:end],
		})
	}
	return chunks
}
