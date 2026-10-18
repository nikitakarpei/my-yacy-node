package pebblevault

import (
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type memberSetFold struct {
	presence memberPresence
}

func memberSetFoldFrom(operand []byte) (*memberSetFold, error) {
	fold := &memberSetFold{presence: memberPresence{}}

	return fold, fold.MergeNewer(operand)
}

func (f *memberSetFold) MergeNewer(operand []byte) error {
	change, err := vault.MemberSetChangeFrom(operand)
	if err != nil {
		return err
	}
	for _, member := range change.Added {
		f.presence[member] = true
	}
	for _, member := range change.Removed {
		f.presence[member] = false
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
		if _, decided := f.presence[member]; !decided {
			f.presence[member] = isPresent
		}
	}
}

func (f *memberSetFold) Finish(includesBase bool) ([]byte, io.Closer, error) {
	change := f.presence.changeOf()
	if includesBase {
		change.Removed = nil
	}

	return change.Encoded(), nil, nil
}
