package undeliveredpostings

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type UndeliveredWordPartition struct {
	WordPartition yacymodel.WordPartition
	Postings      []UndeliveredPosting
}

func (w UndeliveredWordPartition) LastPosting() UndeliveredPosting {
	return w.Postings[len(w.Postings)-1]
}

type wordPartitionReading struct {
	postingsBudget         int
	amountOfPostingsRead   int
	finishedWordPartitions []UndeliveredWordPartition
	openWordPartition      UndeliveredWordPartition
	stoppedOnBudget        bool
}

func wordPartitionReadingWithin(postingsBudget int) *wordPartitionReading {
	return &wordPartitionReading{postingsBudget: postingsBudget}
}

func (r *wordPartitionReading) visit(posting UndeliveredPosting) bool {
	if len(r.openWordPartition.Postings) > 0 &&
		posting.WordPartition != r.openWordPartition.WordPartition {
		r.finishedWordPartitions = append(r.finishedWordPartitions, r.openWordPartition)
		r.openWordPartition = UndeliveredWordPartition{}
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

func (r *wordPartitionReading) wordPartitions() []UndeliveredWordPartition {
	if len(r.openWordPartition.Postings) == 0 {
		return r.finishedWordPartitions
	}

	return append(r.finishedWordPartitions, r.openWordPartition)
}
