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
}

func (s *Schedule) AdmitterFrom(
	admitter rwipostings.PostingAdmitter,
) rwipostings.PostingAdmitter {
	return schedulingAdmitter{admitter: admitter, schedule: s, partitions: s.partitions}
}

func (a schedulingAdmitter) Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if err := a.admitter.Admit(tx, posting); err != nil {
		return err
	}

	return a.schedule.MakeDue(tx, yacymodel.WordPartitionOfPosting(posting, a.partitions))
}
