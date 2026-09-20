package pagecontents

type LinkCounts struct {
	LocalLinks    int
	ExternalLinks int
}

func (counts LinkCounts) AmountOfLinks() int {
	return counts.LocalLinks + counts.ExternalLinks
}
