// Package pagevisitclaim names who holds the claim on a pending page visit
// after a message asked the ledger for it.
package pagevisitclaim

type Claim string

const (
	Unanswered    Claim = ""
	Taken         Claim = "taken"
	Resumed       Claim = "resumed"
	HeldElsewhere Claim = "held-elsewhere"
)
