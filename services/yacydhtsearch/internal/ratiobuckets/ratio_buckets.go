// Package ratiobuckets gives the histogram buckets of a ratio: one bucket for
// each tenth of the whole, from none to all.
package ratiobuckets

const (
	amountOfTenths = 11
	tenth          = 0.1
)

func Tenths() []float64 {
	buckets := make([]float64, 0, amountOfTenths)
	for step := range amountOfTenths {
		buckets = append(buckets, float64(step)*tenth)
	}

	return buckets
}
