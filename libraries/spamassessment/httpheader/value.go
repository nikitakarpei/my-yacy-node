// Package httpheader writes a spam assessment as a Spam-Assessment header
// value, an RFC 9651 Item whose token is the verdict, and reads the verdict
// back from it.
package httpheader

import (
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
)

const Name = "Spam-Assessment"

var tokenPerVerdict = map[spamassessment.Verdict]string{
	spamassessment.Clean: "clean",
	spamassessment.Spam:  "spam",
}

var verdictPerToken = map[string]spamassessment.Verdict{
	"clean": spamassessment.Clean,
	"spam":  spamassessment.Spam,
}

func ValueOf(assessment spamassessment.Assessment) string {
	return tokenPerVerdict[assessment.Verdict()] +
		";score=" + strconv.FormatFloat(assessment.Score, 'f', 3, 64) +
		";threshold=" + strconv.FormatFloat(assessment.Threshold, 'g', -1, 64) +
		";model=" + strconv.Quote(assessment.ModelVersion)
}

func VerdictFrom(spamAssessmentValue string) spamassessment.Verdict {
	token, _, _ := strings.Cut(spamAssessmentValue, ";")

	return verdictPerToken[token]
}
