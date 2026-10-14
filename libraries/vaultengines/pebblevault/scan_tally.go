package pebblevault

import "sync/atomic"

type scanTally struct {
	amounts       atomic.Int64
	amountsAtZero atomic.Int64
}

func (t *scanTally) count(amount int64) {
	t.amounts.Add(1)
	if countFrom(amount) == 0 {
		t.amountsAtZero.Add(1)
	}
}
