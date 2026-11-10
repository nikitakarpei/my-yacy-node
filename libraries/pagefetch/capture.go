package pagefetch

type Capture struct {
	Statement   []byte
	Signature   []byte
	KeyEvidence []byte
}

func (c Capture) Present() bool {
	return len(c.Statement) > 0
}
