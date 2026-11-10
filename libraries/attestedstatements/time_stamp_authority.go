package attestedstatements

type TimeStampAuthority interface {
	Stamp(signedStatementID []byte) ([]byte, error)
}
