package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type URLWordsMetrics struct {
	addedWords       *prometheus.CounterVec
	undecodableWords prometheus.Counter
}

func NewURLWordsMetrics(registry prometheus.Registerer) *URLWordsMetrics {
	addedWords := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "yacynode_url_words_added_total",
		Help: "Words added to the word sets of urls, by form: id for a word the ranking lists, raw for a full word hash.",
	}, []string{"form"})
	undecodableWords := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "yacynode_url_words_undecodable_total",
		Help: "Words read from the word sets of urls that the node could not decode.",
	})
	registry.MustRegister(addedWords, undecodableWords)
	for _, form := range []vault.MemberForm{vault.MemberFormID, vault.MemberFormRaw} {
		addedWords.WithLabelValues(string(form))
	}

	return &URLWordsMetrics{addedWords: addedWords, undecodableWords: undecodableWords}
}

func (m *URLWordsMetrics) ObserveMemberAdded(_ vault.Name, form vault.MemberForm) {
	m.addedWords.WithLabelValues(string(form)).Inc()
}

func (m *URLWordsMetrics) ObserveUndecodableMembers(_ vault.Name, amount int) {
	m.undecodableWords.Add(float64(amount))
}

var _ vault.MemberSetObserver = (*URLWordsMetrics)(nil)
