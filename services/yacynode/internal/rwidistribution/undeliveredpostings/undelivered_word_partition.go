package undeliveredpostings

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type WordPartition struct {
	WordPartition yacymodel.WordPartition
	Postings      []Posting
}

func (w WordPartition) LastPosting() Posting {
	return w.Postings[len(w.Postings)-1]
}

type wordPartitionReading struct {
	postingsBudget         int
	amountOfPostingsRead   int
	finishedWordPartitions []WordPartition
	openWordPartition      WordPartition
	stoppedOnBudget        bool
}

func wordPartitionReadingWithin(postingsBudget int) *wordPartitionReading {
	return &wordPartitionReading{postingsBudget: postingsBudget}
}

func (r *wordPartitionReading) visit(posting Posting) bool {
	if len(r.openWordPartition.Postings) > 0 &&
		posting.WordPartition != r.openWordPartition.WordPartition {
		r.finishedWordPartitions = append(r.finishedWordPartitions, r.openWordPartition)
		r.openWordPartition = WordPartition{}
		if r.amountOfPostingsRead >= r.postingsBudget {
			r.stoppedOnBudget = true

			return false
		}
	}
	r.openWordPartition.WordPartition = posting.WordPartition
	r.openWordPartition.Postings = append(r.openWordPartition.Postings, posting)
	r.amountOfPostingsRead++

	return true
}

func (r *wordPartitionReading) wordPartitions() []WordPartition {
	if len(r.openWordPartition.Postings) == 0 {
		return r.finishedWordPartitions
	}

	return append(r.finishedWordPartitions, r.openWordPartition)
}
