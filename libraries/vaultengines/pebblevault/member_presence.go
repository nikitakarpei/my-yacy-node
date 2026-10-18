package pebblevault

import "github.com/nikitakarpei/yacy-rwi-node/vault"

type memberPresence map[vault.StoredMember]bool

func (p memberPresence) changeOf() vault.MemberSetChange {
	var change vault.MemberSetChange
	for member, isPresent := range p {
		if isPresent {
			change.Added = append(change.Added, member)
		} else {
			change.Removed = append(change.Removed, member)
		}
	}

	return change
}
