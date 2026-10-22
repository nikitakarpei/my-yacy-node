package vault

import "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"

type crawlOrderCodec struct{}

func (crawlOrderCodec) Encode(order yacycrawlcontract.CrawlOrder) ([]byte, error) {
	return yacycrawlcontract.MarshalCrawlOrder(order)
}

func (crawlOrderCodec) Decode(stored []byte) (yacycrawlcontract.CrawlOrder, error) {
	return yacycrawlcontract.UnmarshalCrawlOrder(stored)
}
