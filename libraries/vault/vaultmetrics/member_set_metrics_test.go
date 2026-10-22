package vaultmetrics_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultmetrics"
)

func TestVaultMemberSetsCountTheAddedMembersByBucketAndForm(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := vaultmetrics.NewMemberSetMetrics(registry, "service")

	observer.ObserveMemberAdded("words", vault.MemberFormID)
	observer.ObserveMemberAdded("words", vault.MemberFormID)
	observer.ObserveMemberAdded("words", vault.MemberFormRaw)
	observer.ObserveMemberAdded("hosts", vault.MemberFormRaw)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP service_vault_member_set_members_added_total Members added to the sets of each vault member sets bucket, by form: id for a value the dictionary lists, raw for a full value.
# TYPE service_vault_member_set_members_added_total counter
service_vault_member_set_members_added_total{bucket="hosts",form="raw"} 1
service_vault_member_set_members_added_total{bucket="words",form="id"} 2
service_vault_member_set_members_added_total{bucket="words",form="raw"} 1
`), "service_vault_member_set_members_added_total"); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}

func TestVaultMemberSetsCountTheMembersVaultCouldNotDecodeByBucket(t *testing.T) {
	registry := prometheus.NewRegistry()
	observer := vaultmetrics.NewMemberSetMetrics(registry, "service")

	observer.ObserveUndecodableMembers("words", 3)
	observer.ObserveUndecodableMembers("words", 2)
	observer.ObserveUndecodableMembers("hosts", 1)

	if err := testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP service_vault_member_set_undecodable_members_total Members read from the sets of each vault member sets bucket that vault could not decode.
# TYPE service_vault_member_set_undecodable_members_total counter
service_vault_member_set_undecodable_members_total{bucket="hosts"} 1
service_vault_member_set_undecodable_members_total{bucket="words"} 5
`), "service_vault_member_set_undecodable_members_total"); err != nil {
		t.Fatalf("GatherAndCompare: %v", err)
	}
}
