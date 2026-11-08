package devicepermissions_test

import (
	"slices"
	"testing"

	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/devicepermissions"
)

func TestADeviceMayDoWhatEachPageServiceNeeds(t *testing.T) {
	granted := devicepermissions.ForPageServices()

	for name, needed := range map[string]natspermissions.Permissions{
		"pagescrape": pagescrapeassembly.ServiceConfig{
			ScrapeRequestDurable: pagescrapeassembly.DefaultScrapeRequestDurable,
		}.BrokerPermissions(),
		"pagerwi": pagerwiassembly.ServiceConfig{
			PageOfferDurable: pagerwiassembly.DefaultPageOfferDurable,
		}.BrokerPermissions(),
	} {
		for _, subject := range needed.Publish {
			if !slices.Contains(granted.Publish, subject) {
				t.Errorf("%s may not publish on %s", name, subject)
			}
		}
		for _, subject := range needed.Subscribe {
			if !slices.Contains(granted.Subscribe, subject) {
				t.Errorf("%s may not subscribe to %s", name, subject)
			}
		}
	}
}
