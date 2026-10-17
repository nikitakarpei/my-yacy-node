package main

import (
	"sync/atomic"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
)

type postingAdmissionRelay struct {
	admission atomic.Pointer[rwiadmission.PostingAdmission]
}

func (r *postingAdmissionRelay) forwardTo(admission *rwiadmission.PostingAdmission) {
	r.admission.Store(admission)
}

func (r *postingAdmissionRelay) URLStored(
	tx *vault.Txn,
	url yacymodel.URLHash,
	storedFingerprint pagefingerprint.Fingerprint,
) error {
	if admission := r.admission.Load(); admission != nil {
		return admission.URLStored(tx, url, storedFingerprint)
	}

	return nil
}

func (r *postingAdmissionRelay) URLPurged(*vault.Txn, yacymodel.URLHash) error {
	return nil
}
