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
	for _, word := range pagePostings.Words() {
		wordPartition := yacymodel.WordPartitionOfWordAndURL(word, pagePostings.URL(), a.partitions)
		if err := a.countAdmission(tx, wordPartition); err != nil {
			return err
		}
	}

	return nil
}

func (a schedulingWriter) countAdmission(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) error {
	if err := a.admissions.count(tx, wordPartition); err != nil {
		return err
	}
	position := yacymodel.DHTRingPositionOfWordPartition(wordPartition, a.partitions)
	tx.RunAfterCommit(func() { a.observer.ObservePostingAdmitted(position) })

	return nil
}
