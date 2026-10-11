// Package pagevisitdrop names why a pending page visit was dropped without a visit to
// its page.
package pagevisitdrop

type Reason string

const (
	TakenByAnother     Reason = "taken-by-another"
	NotDue             Reason = "not-due"
	HostPagesExhausted Reason = "host-pages-exhausted"
	DeferralsExhausted Reason = "deferrals-exhausted"
	RetriesExhausted   Reason = "retries-exhausted"
)
