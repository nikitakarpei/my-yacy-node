package judgedqueries_test

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"

type orderedQuery struct {
	judgedQuery      judgedQuery
	orderedDocuments []queryanswers.FoundDocument
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
