package yacymodel

// RFC 6761, RFC 7686, RFC 9476 and ICANN withhold these from the root zone, so they never resolve.
// A delegated top-level domain needs no entry: it maps to the generic domain id, which DomainID
// already returns for any dotted host. Only a withheld one changes an answer, so this set stands in
// for the root zone table YaCy carries to spare itself a lookup this port never makes.
var specialUseTLD = map[string]struct{}{
	"alt":       {},
	"example":   {},
	"internal":  {},
	"invalid":   {},
	"local":     {},
	"localhost": {},
	"onion":     {},
	"test":      {},
}

func isSpecialUseTLD(tld string) bool {
	_, reserved := specialUseTLD[tld]

	return reserved
}
