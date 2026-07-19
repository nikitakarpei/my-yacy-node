package urlmetastaleness

// freshnessRankCodec stores a rank as the plain text it already is, so the
// vault's byte order stays stalest-first.
type freshnessRankCodec struct{}

func (freshnessRankCodec) Encode(rank freshnessRank) ([]byte, error) {
	return []byte(rank), nil
}

func (freshnessRankCodec) Decode(raw []byte) (freshnessRank, error) {
	return freshnessRank(raw), nil
}
