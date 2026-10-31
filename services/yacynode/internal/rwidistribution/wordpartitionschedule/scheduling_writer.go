package wordpartitionschedule

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type schedulingWriter struct {
	writer     rwipostings.PostingWriter
	admissions *admissions
	partitions yacymodel.DHTRingPartitions
	observer   AdmissionObserver
}

func (s *Schedule) WriterFrom(
	writer rwipostings.PostingWriter,
	observer AdmissionObserver,
) rwipostings.PostingWriter {
	return schedulingWriter{
		writer:     writer,
		admissions: s.admissions,
		partitions: s.partitions,
		observer:   observer,
	}
}

func (a schedulingWriter) Write(tx *vault.Txn, pagePostings yacymodel.PagePostings) error {
	if err := a.writer.Write(tx, pagePostings); err != nil {
		return err
	}
	for _, posting := range pagePostings.Postings() {
		if err := a.countAdmission(tx, posting); err != nil {
			return err
		}
	}

	return nil
}

func (a schedulingWriter) countAdmission(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	wordPartition := yacymodel.WordPartitionOfPosting(posting, a.partitions)
	if err := a.admissions.count(tx, wordPartition); err != nil {
		return err
	}
	position := yacymodel.DHTRingPositionOfWordPartition(wordPartition, a.partitions)
	tx.RunAfterCommit(func() { a.observer.ObservePostingAdmitted(position) })

	return nil
}
