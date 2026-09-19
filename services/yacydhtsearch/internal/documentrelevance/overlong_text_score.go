package documentrelevance

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const amountOfWordsBeyondWhichATextIsOverlong = 20000

func overlongTextScoreOf(foundDocument queryanswers.FoundDocument) float64 {
	if foundDocument.AmountOfWords <= amountOfWordsBeyondWhichATextIsOverlong {
		return 0
	}

	return math.Log10(
		float64(foundDocument.AmountOfWords) / amountOfWordsBeyondWhichATextIsOverlong,
	)
}
