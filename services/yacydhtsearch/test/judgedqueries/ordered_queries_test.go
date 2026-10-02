package judgedqueries_test

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"

type orderedQuery struct {
	judgedQuery      judgedQuery
	orderedDocuments []queryfindings.FoundDocument
}

type orderedQueries []orderedQuery

func (queries orderedQueries) gainPerQuery() gainPerQuery {
	gain := make(gainPerQuery, len(queries))
	for _, orderedQuery := range queries {
		gain[orderedQuery.judgedQuery.query] = orderedQuery.judgedQuery.gradedDocuments.
			normalizedGainOf(orderedQuery.orderedDocuments)
	}

	return gain
}
