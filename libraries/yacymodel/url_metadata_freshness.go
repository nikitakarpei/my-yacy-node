package yacymodel

const (
	ColLoadDate  = "load"
	ColModDate   = "mod"
	ColFreshDate = "fresh"
)

var freshnessPrecedence = []string{ColLoadDate, ColModDate, ColFreshDate}

// TODO: freshness picks a wire column and returns its undecoded string instead
// of a time, blocked by URIMetadataRow keeping its properties unparsed.
func (r URIMetadataRow) Freshness() string {
	for _, key := range freshnessPrecedence {
		if value := r.Properties[key]; value != "" {
			return value
		}
	}

	return ""
}
