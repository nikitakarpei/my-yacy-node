package pageservices

type TimeStampAuthority interface {
	Stamp(signedStatementID []byte) ([]byte, error)
}
