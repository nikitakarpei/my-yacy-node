package wordpartitionschedule

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type AdmissionObserver interface {
	ObservePostingAdmitted(position yacymodel.DHTRingPosition)
}
