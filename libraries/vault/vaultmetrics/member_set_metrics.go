package vaultmetrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type MemberSetMetrics struct {
	addedMembers       *prometheus.CounterVec
	undecodableMembers *prometheus.CounterVec
}

func NewMemberSetMetrics(
	registry prometheus.Registerer,
	serviceMetricNamespace string,
) *MemberSetMetrics {
	addedMembers := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: serviceMetricNamespace + "_vault_member_set_members_added_total",
		Help: "Members added to the sets of each vault member sets bucket, by form: id for a value the dictionary lists, raw for a full value.",
	}, []string{labelBucket, "form"})
	undecodableMembers := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: serviceMetricNamespace + "_vault_member_set_undecodable_members_total",
		Help: "Members read from the sets of each vault member sets bucket that vault could not decode.",
	}, []string{labelBucket})
	registry.MustRegister(addedMembers, undecodableMembers)

	return &MemberSetMetrics{
		addedMembers:       addedMembers,
		undecodableMembers: undecodableMembers,
	}
}

func (m *MemberSetMetrics) ObserveMemberAdded(bucket vault.Name, form vault.MemberForm) {
	m.addedMembers.WithLabelValues(string(bucket), string(form)).Inc()
}

func (m *MemberSetMetrics) ObserveUndecodableMembers(bucket vault.Name, amount int) {
	m.undecodableMembers.WithLabelValues(string(bucket)).Add(float64(amount))
}

var _ vault.MemberSetObserver = (*MemberSetMetrics)(nil)
