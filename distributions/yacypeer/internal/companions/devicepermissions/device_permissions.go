// Package devicepermissions holds what a paired device may do on the broker of the peer to run
// the page services: pagescrape and pagerwi with their default consumers.
package devicepermissions

import (
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
)

func ForPageServices() natspermissions.Permissions {
	return natspermissions.JoinedFrom(
		pagescrapeassembly.ServiceConfig{
			ScrapeRequestDurable: pagescrapeassembly.DefaultScrapeRequestDurable,
		}.BrokerPermissions(),
		pagerwiassembly.ServiceConfig{
			PageOfferDurable: pagerwiassembly.DefaultPageOfferDurable,
		}.BrokerPermissions(),
	)
}
