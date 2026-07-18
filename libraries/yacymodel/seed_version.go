package yacymodel

type YaCyVersion string

// TODO: a parser that cannot fail states a validation rule this type does not
// have; nothing blocks dropping the error once callers stop threading it.
func ParseYaCyVersion(s string) (YaCyVersion, error) {
	return YaCyVersion(s), nil
}

func (v YaCyVersion) String() string {
	return string(v)
}
