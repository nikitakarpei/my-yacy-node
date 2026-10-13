package pebblevault

type heldValues map[string][]byte

func (h heldValues) valueAt(key []byte) ([]byte, bool) {
	value, held := h[string(key)]

	return value, held
}

func (h heldValues) hold(key, value []byte) {
	if h == nil {
		return
	}
	h[string(key)] = value
}

func (h heldValues) forget(key []byte) {
	delete(h, string(key))
}
