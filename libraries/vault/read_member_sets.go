package vault

import "fmt"

type readMemberSets struct {
	bucket Name
	sets   map[string]*readMemberSet
}

type readMemberSet struct {
	unremovedMembers map[StoredMember]struct{}
	wasAdded         bool
}

func newReadMemberSets(bucket Name) *readMemberSets {
	return &readMemberSets{bucket: bucket, sets: map[string]*readMemberSet{}}
}

func (r *readMemberSets) membersRead(memberSetKey []byte, members []StoredMember) {
	if len(members) == 0 {
		return
	}
	readSet, found := r.sets[string(memberSetKey)]
	if !found {
		readSet = &readMemberSet{unremovedMembers: map[StoredMember]struct{}{}}
		r.sets[string(memberSetKey)] = readSet
	}
	for _, member := range members {
		readSet.unremovedMembers[member] = struct{}{}
	}
}

func (r *readMemberSets) memberAdded(memberSetKey []byte) {
	if readSet, found := r.sets[string(memberSetKey)]; found {
		readSet.wasAdded = true
	}
}

func (r *readMemberSets) memberRemoved(memberSetKey []byte, member StoredMember) {
	if readSet, found := r.sets[string(memberSetKey)]; found {
		delete(readSet.unremovedMembers, member)
	}
}

func (r *readMemberSets) finishWithin(etx EngineTxn) error {
	for memberSetKey, readSet := range r.sets {
		if readSet.wasAdded || len(readSet.unremovedMembers) > 0 {
			continue
		}
		if err := etx.MemberSets(r.bucket).Delete([]byte(memberSetKey)); err != nil {
			return fmt.Errorf("delete emptied %s set: %w", r.bucket, err)
		}
	}

	return nil
}
