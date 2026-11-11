package attestedstatements

type Bundle []byte

func (b Bundle) MarshalJSON() ([]byte, error) {
	if len(b) == 0 {
		return []byte("null"), nil
	}

	return b, nil
}

func (b *Bundle) UnmarshalJSON(encoding []byte) error {
	*b = append(Bundle(nil), encoding...)

	return nil
}
