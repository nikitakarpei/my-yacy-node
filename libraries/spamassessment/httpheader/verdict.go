// Package httpheader reads the spam verdict of a page from its Spam-Assessment
// header value, an RFC 9651 Item whose token is the verdict.
package httpheader

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
)

var verdictPerToken = map[string]spamassessment.Verdict{
	"clean": spamassessment.Clean,
	"spam":  spamassessment.Spam,
}

func VerdictFrom(spamAssessmentValue string) spamassessment.Verdict {
	token, _, _ := strings.Cut(spamAssessmentValue, ";")

	return verdictPerToken[token]
}
