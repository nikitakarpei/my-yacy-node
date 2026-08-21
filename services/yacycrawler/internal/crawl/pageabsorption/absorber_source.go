package pageabsorption

type AbsorberSource interface {
	AbsorberFor(indexingRefusal IndexingRefusal) Absorber
}

type absorberSource struct {
	extractor PageExtractor
}

func New(extractor PageExtractor) AbsorberSource {
	return &absorberSource{extractor: extractor}
}

func (s *absorberSource) AbsorberFor(indexingRefusal IndexingRefusal) Absorber {
	return &absorber{
		extractor:       s.extractor,
		indexingRefusal: indexingRefusal,
	}
}
