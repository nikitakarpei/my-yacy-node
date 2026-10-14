package nodestatus

import (
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type wordAmountCache struct {
	words        WordAmountEstimate
	now          func() time.Time
	lifetime     time.Duration
	mutex        sync.Mutex
	latestAmount int
	expiresAt    time.Time
}

func (c *wordAmountCache) EstimatedAmountOfWords(tx *vault.Txn) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.now().Before(c.expiresAt) {
		return c.latestAmount, nil
	}
	amountOfWords, err := c.words.EstimatedAmountOfWords(tx)
	if err != nil {
		return 0, err
	}
	c.latestAmount, c.expiresAt = amountOfWords, c.now().Add(c.lifetime)

	return amountOfWords, nil
}
