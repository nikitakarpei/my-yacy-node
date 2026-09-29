// Package spamassessment holds the verdict that a spam assessment gives a page.
// Its subpackages read the verdict from one carrier each.
package spamassessment

type Verdict int

const (
	Unassessed Verdict = iota
	Clean
	Spam
)
