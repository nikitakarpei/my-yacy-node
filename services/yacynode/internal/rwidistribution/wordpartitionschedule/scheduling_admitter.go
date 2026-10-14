package wordpartitionschedule

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type schedulingAdmitter struct {
	admitter   rwipostings.PostingAdmitter
	schedule   *Schedule
	partitions yacymodel.DHTRingPartitions
	observer   DueObserver
}

func (s *Schedule) AdmitterFrom(
	admitter rwipostings.PostingAdmitter,
	observer DueObserver,
) rwipostings.PostingAdmitter {
	return schedulingAdmitter{
		admitter:   admitter,
		schedule:   s,
		partitions: s.partitions,
		observer:   observer,
	}
}

func (a schedulingAdmitter) Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if err := a.admitter.Admit(tx, posting); err != nil {
		return err
	}
	wordPartition := yacymodel.WordPartitionOfPosting(posting, a.partitions)
	wasMadeDue, err := a.schedule.MakeDue(tx, wordPartition)
	if err != nil || !wasMadeDue {
		return err
	}
	position := yacymodel.DHTRingPositionOfWordPartition(wordPartition, a.partitions)
	tx.RunAfterCommit(func() { a.observer.ObserveWordPartitionDue(position) })

	return nil
}
