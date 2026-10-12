package pebblevault

type readFootprint struct {
	keys   map[string]heldValue
	ranges []visitedRange
}

type heldValue struct {
	value []byte
	known bool
}

type visitedRange struct {
	firstIncluded []byte
	firstExcluded []byte
}

func newReadFootprint() *readFootprint {
	return &readFootprint{keys: map[string]heldValue{}}
}

func (f *readFootprint) valueKnownAt(key []byte) ([]byte, bool) {
	if f == nil {
		return nil, false
	}
	held := f.keys[string(key)]

	return held.value, held.known
}

func (f *readFootprint) addKeyHolding(key, value []byte) {
	if f == nil {
		return
	}
	f.keys[string(key)] = heldValue{value: value, known: true}
}

func (f *readFootprint) forgetValueAt(key []byte) {
	if f == nil {
		return
	}
	if _, read := f.keys[string(key)]; read {
		f.keys[string(key)] = heldValue{}
	}
}

func (f *readFootprint) addRange(firstIncluded, firstExcluded []byte) {
	if f == nil {
		return
	}
	f.ranges = append(
		f.ranges,
		visitedRange{firstIncluded: firstIncluded, firstExcluded: firstExcluded},
	)
}

func (f *readFootprint) readKeyAmong(writtenKeys sortedKeys) ([]byte, bool) {
	for _, key := range writtenKeys {
		if _, read := f.keys[string(key)]; read {
			return key, true
		}
	}
	for _, visited := range f.ranges {
		if key, found := writtenKeys.firstWithin(visited); found {
			return key, true
		}
	}

	return nil, false
}
