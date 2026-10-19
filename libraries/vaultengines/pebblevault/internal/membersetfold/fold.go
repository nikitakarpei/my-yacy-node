// Package membersetfold encodes the changes that merge operands carry for one
// member set and folds them into one change. A newer change to a member wins over
// an older one. A fold that reaches the set's base keeps only the members present.
package membersetfold

import (
	"io"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/memberset"
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

const operandsBeforeGrowth = 4

type Fold struct {
	operands      []foldedOperand
	firstOperands [operandsBeforeGrowth]foldedOperand
	newestAge     int
	oldestAge     int
}

type foldedOperand struct {
	encoded        []byte
	age            int
	addedMembers   memberset.Decoder
	removedMembers memberset.Decoder
}

func From(operand []byte) (*Fold, error) {
	fold := &Fold{}
	fold.operands = fold.firstOperands[:0]

	return fold, fold.hold(operand, 0)
}

func (f *Fold) hold(operand []byte, age int) error {
	encoded := slices.Clone(operand)
	addedMembers, removedMembers, err := decodersFrom(encoded)
	if err != nil {
		return err
	}
	f.operands = append(f.operands, foldedOperand{
		encoded:        encoded,
		age:            age,
		addedMembers:   addedMembers,
		removedMembers: removedMembers,
	})

	return nil
}

func (f *Fold) MergeNewer(operand []byte) error {
	f.newestAge--

	return f.hold(operand, f.newestAge)
}

func (f *Fold) MergeOlder(operand []byte) error {
	f.oldestAge++

	return f.hold(operand, f.oldestAge)
}

func (f *Fold) Finish(includesBase bool) ([]byte, io.Closer, error) {
	if len(f.operands) == 1 &&
		(!includesBase || f.operands[0].removedMembers.AmountOfMembers() == 0) {
		return f.operands[0].encoded, nil, nil
	}

	return f.folded(includesBase), nil, nil
}

func (f *Fold) folded(includesBase bool) []byte {
	cursors := cursorsOver(f.operands)
	var added, removed memberset.Encoder
	for member, found := cursors.lowest(); found; member, found = cursors.lowest() {
		switch {
		case cursors.takePresenceOf(member):
			added.Add(member)
		case !includesBase:
			removed.Add(member)
		}
	}

	return encodedChangeOf(added, removed)
}

type memberCursor struct {
	decoder   memberset.Decoder
	member    memberset.DecodedMember
	isValid   bool
	isPresent bool
	age       int
}

type memberCursors []memberCursor

func cursorsOver(operands []foldedOperand) memberCursors {
	cursors := make(memberCursors, 0, 2*len(operands))
	for _, operand := range operands {
		cursors = cursors.withCursorOver(operand.addedMembers, true, operand.age)
		cursors = cursors.withCursorOver(operand.removedMembers, false, operand.age)
	}

	return cursors
}

func (c memberCursors) withCursorOver(
	decoder memberset.Decoder,
	isPresent bool,
	age int,
) memberCursors {
	cursor := memberCursor{decoder: decoder, isPresent: isPresent, age: age}
	cursor.advance()
	if !cursor.isValid {
		return c
	}

	return append(c, cursor)
}

func (c *memberCursor) advance() {
	c.member, c.isValid = c.decoder.Next()
}

func (c memberCursors) lowest() (memberset.DecodedMember, bool) {
	var lowestMember memberset.DecodedMember
	found := false
	for index := range c {
		cursor := &c[index]
		if cursor.isValid && (!found || cursor.member.CompareInStoredOrder(lowestMember) < 0) {
			lowestMember, found = cursor.member, true
		}
	}

	return lowestMember, found
}

func (c memberCursors) takePresenceOf(member memberset.DecodedMember) bool {
	var newestAge int
	isPresent, found := false, false
	for index := range c {
		cursor := &c[index]
		if !cursor.isValid || cursor.member.CompareInStoredOrder(member) != 0 {
			continue
		}
		if !found || cursor.age < newestAge {
			newestAge, isPresent, found = cursor.age, cursor.isPresent, true
		}
		cursor.advance()
	}

	return isPresent
}
