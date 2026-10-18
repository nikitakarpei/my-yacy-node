package vault

import "fmt"

type memberSetReads map[memberSetPlace]*memberSetRead

type memberSetPlace struct {
	bucket   Name
	groupKey string
}

type memberSetRead struct {
	unremovedMembers map[StoredMember]struct{}
	wasAdded         bool
}

func (r memberSetReads) read(bucket Name, groupKey []byte, members []StoredMember) {
	if r == nil || len(members) == 0 {
		return
	}
	place := memberSetPlace{bucket: bucket, groupKey: string(groupKey)}
	read, found := r[place]
	if !found {
		read = &memberSetRead{unremovedMembers: map[StoredMember]struct{}{}}
		r[place] = read
	}
	for _, member := range members {
		read.unremovedMembers[member] = struct{}{}
	}
}

func (r memberSetReads) memberAdded(bucket Name, groupKey []byte) {
	if read, found := r[memberSetPlace{bucket: bucket, groupKey: string(groupKey)}]; found {
		read.wasAdded = true
	}
}

func (r memberSetReads) memberRemoved(bucket Name, groupKey []byte, member StoredMember) {
	if read, found := r[memberSetPlace{bucket: bucket, groupKey: string(groupKey)}]; found {
		delete(read.unremovedMembers, member)
	}
}

func (r memberSetReads) deleteEmptiedWithin(etx EngineTxn) error {
	for place, read := range r {
		if read.wasAdded || len(read.unremovedMembers) > 0 {
			continue
		}
		if err := etx.MemberSets(place.bucket).Clear([]byte(place.groupKey)); err != nil {
			return fmt.Errorf("delete emptied %s set: %w", place.bucket, err)
		}
	}

	return nil
}
