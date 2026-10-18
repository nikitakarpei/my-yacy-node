package vault

import "strconv"

type StoredMember struct {
	ID  uint64
	Raw string
}

type MemberForm string

const (
	MemberFormID  MemberForm = "id"
	MemberFormRaw MemberForm = "raw"
)

func (m StoredMember) Form() MemberForm {
	if m.Raw != "" {
		return MemberFormRaw
	}

	return MemberFormID
}

func (m StoredMember) identity() string {
	if m.Raw != "" {
		return "r" + m.Raw
	}

	return "i" + strconv.FormatUint(m.ID, 10)
}
