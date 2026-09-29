// Package spamassessment holds the spam assessment of a page and the verdict
// that it gives. Its subpackages carry the assessment in one medium each.
package spamassessment

type Verdict int

const (
	Unassessed Verdict = iota
	Clean
	Spam
)

var verdictNames = map[Verdict]string{Unassessed: "unassessed", Clean: "clean", Spam: "spam"}

func (v Verdict) String() string {
	return verdictNames[v]
}
