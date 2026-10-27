package pageclassification

type Verdict int

const (
	Unassessed Verdict = iota
	Negative
	Positive
)

var verdictNames = map[Verdict]string{
	Unassessed: "unassessed",
	Negative:   "negative",
	Positive:   "positive",
}

func (v Verdict) String() string {
	return verdictNames[v]
}

type Verdicts map[Category]Verdict
