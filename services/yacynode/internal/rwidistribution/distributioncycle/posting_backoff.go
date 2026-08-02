package distributioncycle

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
)

type postingBackoff struct {
	longest map[postingschedule.Identity]time.Duration
}

func newPostingBackoff() *postingBackoff {
	return &postingBackoff{
		longest: make(map[postingschedule.Identity]time.Duration),
	}
}

func (b *postingBackoff) Record(peerOffer offer, backoff time.Duration) {
	for _, posting := range peerOffer.Postings {
		identity := postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash}
		if backoff > b.longest[identity] {
			b.longest[identity] = backoff
		}
	}
}

func (b *postingBackoff) Longest(identity postingschedule.Identity) time.Duration {
	return b.longest[identity]
}
