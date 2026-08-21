package main

import (
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/containerexpanders/archive"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
)

type registeredContainerExpander interface {
	contentextraction.ContainerExpander
	MediaTypes() []string
}

func containerExpanderCatalog(cfg ServiceConfig) []registeredContainerExpander {
	return []registeredContainerExpander{
		archive.New(archiveMaxMembers, cfg.MaxBodyBytes),
	}
}
