// Package pagevisitdrop names why the crawler stopped trying to visit a page.
package pagevisitdrop

type Reason string

const (
	TakenByAnother     Reason = "taken-by-another"
	NotDue             Reason = "not-due"
	HostPagesExhausted Reason = "host-pages-exhausted"
	DeferralsExhausted Reason = "deferrals-exhausted"
	RetriesExhausted   Reason = "retries-exhausted"
	OrderForgotten     Reason = "order-forgotten"
)
