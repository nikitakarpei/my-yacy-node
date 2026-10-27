// Package httpheader writes a page classification as a Page-Classification
// header value, an RFC 9651 Dictionary with one member per category, and reads
// the verdicts back from it. The members come in the order of their scores,
// the highest first.
package httpheader

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

const Name = "Page-Classification"

var valuePerVerdict = map[pageclassification.Verdict]string{
	pageclassification.Positive: "",
	pageclassification.Negative: "=?0",
}

var verdictPerValue = map[string]pageclassification.Verdict{
	"":    pageclassification.Positive,
	"=?1": pageclassification.Positive,
	"=?0": pageclassification.Negative,
}

func ValueOf(classification pageclassification.Classification) string {
	members := make([]string, 0, len(classification))
	for _, category := range categoriesByScoreOf(classification) {
		members = append(members, memberOf(category, classification[category]))
	}
	return strings.Join(members, ", ")
}

func categoriesByScoreOf(
	classification pageclassification.Classification,
) []pageclassification.Category {
	categories := slices.Collect(maps.Keys(classification))
	slices.SortFunc(categories, func(first, second pageclassification.Category) int {
		return cmp.Or(
			cmp.Compare(classification[second].Score, classification[first].Score),
			cmp.Compare(first, second),
		)
	})
	return categories
}

func memberOf(
	category pageclassification.Category,
	assessment pageclassification.Assessment,
) string {
	return string(category) + valuePerVerdict[assessment.Verdict()] +
		";score=" + decimalOf(assessment.Score) +
		";threshold=" + decimalOf(assessment.Threshold) +
		";model=" + strconv.Quote(assessment.ModelVersion)
}

func decimalOf(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func VerdictsFrom(classificationValue string) pageclassification.Verdicts {
	verdicts := pageclassification.Verdicts{}
	for _, member := range partsOf(classificationValue, ',') {
		memberItem := strings.TrimSpace(partsOf(member, ';')[0])
		valueStart := strings.IndexByte(memberItem, '=')
		if valueStart < 0 {
			valueStart = len(memberItem)
		}
		category, categoryErr := pageclassification.CategoryFrom(memberItem[:valueStart])
		verdict, known := verdictPerValue[memberItem[valueStart:]]
		if categoryErr == nil && known {
			verdicts[category] = verdict
		}
	}
	return verdicts
}

func partsOf(value string, separator byte) []string {
	var parts []string
	partStart := 0
	inString := false
	for position := 0; position < len(value); position++ {
		switch {
		case inString && value[position] == '\\':
			position++
		case value[position] == '"':
			inString = !inString
		case !inString && value[position] == separator:
			parts = append(parts, value[partStart:position])
			partStart = position + 1
		}
	}
	return append(parts, value[partStart:])
}
