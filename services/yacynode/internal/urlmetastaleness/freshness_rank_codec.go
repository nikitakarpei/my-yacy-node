package urlmetastaleness

// freshnessRankValueCodec stores a rank as the plain text it already is, so the
// vault's byte order stays stalest-first.
type freshnessRankValueCodec struct{}

func (freshnessRankValueCodec) Encode(rank freshnessRank) ([]byte, error) {
	return []byte(rank), nil
}

func (freshnessRankValueCodec) Decode(raw []byte) (freshnessRank, error) {
	return freshnessRank(raw), nil
}
