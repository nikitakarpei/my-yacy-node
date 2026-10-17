package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type urlStoredObserver interface {
	URLStored(tx *vault.Txn, url yacymodel.URLHash, version urlmeta.CurrentVersion) error
}

// TECHDEBT: Structure — the relay owns no state, endpoint or value type; it only breaks the construction cycle between urlmeta and rwiadmission.
type postingAdmissionRelay struct {
	admission urlStoredObserver
}

func (r *postingAdmissionRelay) forwardTo(admission urlStoredObserver) {
	r.admission = admission
}

func (r *postingAdmissionRelay) URLStored(
	tx *vault.Txn,
	url yacymodel.URLHash,
	version urlmeta.CurrentVersion,
) error {
	if r.admission == nil {
		return nil
	}

	return r.admission.URLStored(tx, url, version)
}

func (r *postingAdmissionRelay) URLPurged(*vault.Txn, yacymodel.URLHash) error {
	return nil
}
