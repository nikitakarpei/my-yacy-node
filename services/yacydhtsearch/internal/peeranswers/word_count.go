package peeranswers

type WordCount struct {
	Hits      int
	TextWords int
}

func (c WordCount) CountedByAPeer() bool {
	return c.Hits > 0 || c.TextWords > 0
}
