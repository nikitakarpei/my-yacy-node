package spamassessment

type Assessment struct {
	Score        float64
	Threshold    float64
	ModelVersion string
}

func (a Assessment) Verdict() Verdict {
	if a.Score > a.Threshold {
		return Spam
	}

	return Clean
}
