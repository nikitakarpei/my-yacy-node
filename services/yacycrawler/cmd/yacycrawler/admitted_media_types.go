package main

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/mediatypeallowance"
)

type admittedMediaTypes struct {
	extractors     map[string]contentextraction.MediaExtractor
	emittedFormats []contentformatgraph.Format
}

func admittedMediaTypesFor(cfg ServiceConfig) (admittedMediaTypes, error) {
	extractors := pagescrape.MediaExtractorCatalog()
	if err := ensureRegisteredMediaTypes(
		cfg.ContentTypes,
		registeredMediaTypes(extractors),
	); err != nil {
		return admittedMediaTypes{}, err
	}
	allowance := mediatypeallowance.MediaTypeAllowanceFrom(cfg.ContentTypes)
	return admittedMediaTypes{
		extractors:     admittedExtractorsFrom(extractors, allowance),
		emittedFormats: emittedFormatsFrom(extractors, allowance),
	}, nil
}

func registeredMediaTypes(extractors []pagescrape.RegisteredMediaExtractor) map[string]bool {
	registered := map[string]bool{}
	for _, extractor := range extractors {
		for _, mediaType := range extractor.MediaTypes() {
			registered[mediaType] = true
		}
	}
	return registered
}

func ensureRegisteredMediaTypes(contentTypes []string, registered map[string]bool) error {
	for _, mediaType := range contentTypes {
		if !registered[mediaType] {
			return fmt.Errorf("%s: no extractor reads %q", EnvContentTypes, mediaType)
		}
	}

	return nil
}

func admittedExtractorsFrom(
	extractors []pagescrape.RegisteredMediaExtractor,
	allowance mediatypeallowance.MediaTypeAllowance,
) map[string]contentextraction.MediaExtractor {
	admitted := map[string]contentextraction.MediaExtractor{}
	for _, extractor := range extractors {
		for _, mediaType := range extractor.MediaTypes() {
			if allowance.Admits(mediaType) {
				admitted[mediaType] = extractor
			}
		}
	}
	return admitted
}

func emittedFormatsFrom(
	extractors []pagescrape.RegisteredMediaExtractor,
	allowance mediatypeallowance.MediaTypeAllowance,
) []contentformatgraph.Format {
	var formats []contentformatgraph.Format
	for _, extractor := range extractors {
		for _, mediaType := range extractor.MediaTypes() {
			if allowance.Admits(mediaType) {
				formats = append(formats, extractor.EmittedFormat())
				break
			}
		}
	}
	return formats
}
