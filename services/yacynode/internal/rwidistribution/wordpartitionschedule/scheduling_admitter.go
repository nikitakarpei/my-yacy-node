package wordpartitionschedule

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type schedulingAdmitter struct {
	admitter   rwipostings.PostingAdmitter
	admissions *admissions
	partitions yacymodel.DHTRingPartitions
	observer   AdmissionObserver
}

func (s *Schedule) AdmitterFrom(
	admitter rwipostings.PostingAdmitter,
	observer AdmissionObserver,
) rwipostings.PostingAdmitter {
	return schedulingAdmitter{
		admitter:   admitter,
		admissions: s.admissions,
		partitions: s.partitions,
		observer:   observer,
	}
}

func (a schedulingAdmitter) Admit(tx *vault.Txn, postings ...yacymodel.RWIPosting) error {
	if err := a.admitter.Admit(tx, postings...); err != nil {
		return err
	}
	for _, posting := range postings {
		if err := a.countAdmission(tx, posting); err != nil {
			return err
		}
	}

	return nil
}

func (a schedulingAdmitter) countAdmission(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	wordPartition := yacymodel.WordPartitionOfPosting(posting, a.partitions)
	if err := a.admissions.count(tx, wordPartition); err != nil {
		return err
	}
	position := yacymodel.DHTRingPositionOfWordPartition(wordPartition, a.partitions)
	tx.RunAfterCommit(func() { a.observer.ObservePostingAdmitted(position) })

	return nil
}
