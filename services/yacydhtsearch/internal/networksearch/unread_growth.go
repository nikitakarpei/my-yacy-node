package networksearch

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"

type unreadGrowth struct{}

func (unreadGrowth) FindingsGrew(queryfindings.Findings) {}
