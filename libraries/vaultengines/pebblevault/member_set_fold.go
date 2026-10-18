package pebblevault

import (
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type memberSetFold struct {
	presenceByMember map[vault.StoredMember]bool
}

func memberSetFoldFrom(operand []byte) (*memberSetFold, error) {
	fold := &memberSetFold{presenceByMember: map[vault.StoredMember]bool{}}

	return fold, fold.MergeNewer(operand)
}

func (f *memberSetFold) MergeNewer(operand []byte) error {
	change, err := vault.MemberSetChangeFrom(operand)
	if err != nil {
		return err
	}
	for _, member := range change.Added {
		f.presenceByMember[member] = true
	}
	for _, member := range change.Removed {
		f.presenceByMember[member] = false
	}

	return nil
}

func (f *memberSetFold) MergeOlder(operand []byte) error {
	change, err := vault.MemberSetChangeFrom(operand)
	if err != nil {
		return err
	}
	f.decideUndecided(change.Added, true)
	f.decideUndecided(change.Removed, false)

	return nil
}

func (f *memberSetFold) decideUndecided(members []vault.StoredMember, isPresent bool) {
	for _, member := range members {
		if _, decided := f.presenceByMember[member]; !decided {
			f.presenceByMember[member] = isPresent
		}
	}
}

func (f *memberSetFold) Finish(includesBase bool) ([]byte, io.Closer, error) {
	var change vault.MemberSetChange
	for member, isPresent := range f.presenceByMember {
		switch {
		case isPresent:
			change.Added = append(change.Added, member)
		case !includesBase:
			change.Removed = append(change.Removed, member)
		}
	}

	return change.Encoded(), nil, nil
}
