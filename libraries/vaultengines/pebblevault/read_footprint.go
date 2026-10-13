package pebblevault

type readFootprint struct {
	keys   map[string]struct{}
	ranges []visitedRange
}

type visitedRange struct {
	firstIncluded []byte
	firstExcluded []byte
}

func newReadFootprint() *readFootprint {
	return &readFootprint{keys: map[string]struct{}{}}
}

func (f *readFootprint) addKey(key []byte) {
	if f == nil {
		return
	}
	f.keys[string(key)] = struct{}{}
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
