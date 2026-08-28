// Package visitclaim names who holds the claim on a pending visit after a
// message asked the ledger for it.
package visitclaim

type Claim string

const (
	Unanswered    Claim = ""
	Taken         Claim = "taken"
	Resumed       Claim = "resumed"
	HeldElsewhere Claim = "held-elsewhere"
)
