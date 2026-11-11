package pagefetch

type ResponseAttestation struct {
	Statement   []byte
	Signature   []byte
	KeyEvidence []byte
}
