package contentformatgraph

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Graph struct {
	byTargetFormat map[crawlcapability.PageContentFormat][]crawlcapability.PageDerivation
}

func New(derivations []crawlcapability.PageDerivation) Graph {
	byTargetFormat := make(
		map[crawlcapability.PageContentFormat][]crawlcapability.PageDerivation,
		len(derivations),
	)
	for _, derivation := range derivations {
		byTargetFormat[derivation.TargetFormat()] = append(
			byTargetFormat[derivation.TargetFormat()],
			derivation,
		)
	}
	return Graph{byTargetFormat: byTargetFormat}
}

func (g Graph) Validate(targetFormats []crawlcapability.PageContentFormat) error {
	reachable := map[crawlcapability.PageContentFormat]bool{}
	var require func(crawlcapability.PageContentFormat) error
	require = func(format crawlcapability.PageContentFormat) error {
		if format == crawlcapability.PageContentFormatDocumentHTML || reachable[format] {
			return nil
		}
		candidates, ok := g.byTargetFormat[format]
		if !ok {
			return fmt.Errorf("%s content is read but no derivation produces it", format)
		}
		reachable[format] = true
		for _, candidate := range candidates {
			if err := require(candidate.SourceFormat()); err != nil {
				return err
			}
		}
		return nil
	}
	for _, format := range targetFormats {
		if err := require(format); err != nil {
			return err
		}
	}
	return nil
}

func (g Graph) Resolver(
	pageURL string,
	format crawlcapability.PageContentFormat,
	body []byte,
) *Resolver {
	return &Resolver{
		pageURL:      pageURL,
		graph:        g,
		contents:     map[crawlcapability.PageContentFormat][]byte{format: body},
		unresolvable: make(map[crawlcapability.PageContentFormat]bool),
		resolving:    make(map[crawlcapability.PageContentFormat]bool),
	}
}
