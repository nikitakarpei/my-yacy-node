// Package direct links each result straight to its address.
package direct

type Direct struct{}

func (Direct) LinkFor(address string) string {
	return address
}
