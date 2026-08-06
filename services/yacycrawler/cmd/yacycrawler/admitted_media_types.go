package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

type admittedMediaTypes struct {
	extractors     map[string]contentextraction.MediaExtractor
	containers     map[string]contentextraction.ContainerExpander
	emittedFormats []contentformatgraph.Format
}

func admittedMediaTypesFor(cfg ServiceConfig) (admittedMediaTypes, error) {
	extractors := mediaExtractorCatalog()
	expanders := containerExpanderCatalog(cfg)
	if err := ensureRegisteredMediaTypes(
		cfg.ContentTypes,
		registeredMediaTypes(extractors, expanders),
	); err != nil {
		return admittedMediaTypes{}, err
	}
	allowance := mediaTypeAllowanceFrom(cfg.ContentTypes)
	return admittedMediaTypes{
		extractors:     admittedExtractorsFrom(extractors, allowance),
		containers:     admittedContainersFrom(expanders, allowance),
		emittedFormats: emittedFormatsFrom(extractors, allowance),
	}, nil
}

func registeredMediaTypes(
	extractors []registeredMediaExtractor,
	expanders []registeredContainerExpander,
) map[string]bool {
	registered := map[string]bool{}
	for _, extractor := range extractors {
		for _, mediaType := range extractor.MediaTypes() {
			registered[mediaType] = true
		}
	}
	for _, expander := range expanders {
		for _, mediaType := range expander.MediaTypes() {
			registered[mediaType] = true
		}
	}
	return registered
}

func admittedExtractorsFrom(
	extractors []registeredMediaExtractor,
	allowance mediaTypeAllowance,
) map[string]contentextraction.MediaExtractor {
	admitted := map[string]contentextraction.MediaExtractor{}
	for _, extractor := range extractors {
		for _, mediaType := range extractor.MediaTypes() {
			if allowance.admits(mediaType) {
				admitted[mediaType] = extractor
			}
		}
	}
	return admitted
}

func admittedContainersFrom(
	expanders []registeredContainerExpander,
	allowance mediaTypeAllowance,
) map[string]contentextraction.ContainerExpander {
	admitted := map[string]contentextraction.ContainerExpander{}
	for _, expander := range expanders {
		for _, mediaType := range expander.MediaTypes() {
			if allowance.admits(mediaType) {
				admitted[mediaType] = expander
			}
		}
	}
	return admitted
}

func emittedFormatsFrom(
	extractors []registeredMediaExtractor,
	allowance mediaTypeAllowance,
) []contentformatgraph.Format {
	var formats []contentformatgraph.Format
	for _, extractor := range extractors {
		for _, mediaType := range extractor.MediaTypes() {
			if allowance.admits(mediaType) {
				formats = append(formats, extractor.EmittedFormat())
				break
			}
		}
	}
	return formats
}
