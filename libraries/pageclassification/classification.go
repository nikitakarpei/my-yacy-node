// Package pageclassification holds the classification of a page: the
// assessment of the page for each category, and the verdicts that the
// assessments give. Its subpackages carry the classification in one medium
// each.
package pageclassification

type Classification map[Category]Assessment
