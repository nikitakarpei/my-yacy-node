package pullintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

var errHeldBack = errors.New("held back for good")

type countedAdmission struct {
	err    error
	awaits *int
}

func (a countedAdmission) AwaitAdmission(context.Context) error {
	*a.awaits++
	return a.err
}

func TestAdmissionOfEachAdmitsOnceEveryAdmissionAdmits(t *testing.T) {
	firstAwaits, secondAwaits := 0, 0

	err := pullintake.AdmissionOfEach(
		countedAdmission{awaits: &firstAwaits}, countedAdmission{awaits: &secondAwaits},
	).AwaitAdmission(context.Background())

	if err != nil || firstAwaits != 1 || secondAwaits != 1 {
		t.Errorf(
			"err = %v, awaits = %d and %d, want both awaited once",
			err,
			firstAwaits,
			secondAwaits,
		)
	}
}

func TestAdmissionOfEachStopsAtTheFirstAdmissionThatEnds(t *testing.T) {
	firstAwaits, secondAwaits := 0, 0

	err := pullintake.AdmissionOfEach(
		countedAdmission{
			awaits: &firstAwaits,
			err:    errHeldBack,
		},
		countedAdmission{awaits: &secondAwaits},
	).AwaitAdmission(context.Background())

	if !errors.Is(err, errHeldBack) || secondAwaits != 0 {
		t.Errorf(
			"err = %v, second awaited %d times, want the first failure only",
			err,
			secondAwaits,
		)
	}
}
