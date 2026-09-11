package peeranswers

type WordCount struct {
	Hits      int
	TextWords int
}

// TECHDEBT: naming — CountedByAPeer names a peer as the only counter, while a
// count now also comes from the text of a document that was read.
func (c WordCount) CountedByAPeer() bool {
	return c.Hits > 0 || c.TextWords > 0
}
