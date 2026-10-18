// Package membersetfold encodes the changes that merge operands carry for one
// member set and folds them into one change. A newer change to a member wins over
// an older one. A fold that reaches the set's base keeps only the members present.
package membersetfold

import (
	"io"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type Presence map[vault.StoredMember]bool

func (p Presence) ChangeOf() Change {
	var change Change
	for member, isPresent := range p {
		if isPresent {
			change.Added = append(change.Added, member)
		} else {
			change.Removed = append(change.Removed, member)
		}
	}

	return change
}

type Fold struct {
	presence Presence
}

func From(operand []byte) (*Fold, error) {
	fold := &Fold{presence: Presence{}}

	return fold, fold.MergeNewer(operand)
}

func (f *Fold) MergeNewer(operand []byte) error {
	change, err := ChangeFrom(operand)
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

func (f *Fold) MergeOlder(operand []byte) error {
	change, err := ChangeFrom(operand)
	if err != nil {
		return err
	}
	f.decideUndecided(change.Added, true)
	f.decideUndecided(change.Removed, false)

	return nil
}

func (f *Fold) decideUndecided(members []vault.StoredMember, isPresent bool) {
	for _, member := range members {
		if _, decided := f.presence[member]; !decided {
			f.presence[member] = isPresent
		}
	}
}

func (f *Fold) Finish(includesBase bool) ([]byte, io.Closer, error) {
	change := f.presence.ChangeOf()
	if includesBase {
		change.Removed = nil
	}

	return change.Encoded(), nil, nil
}
