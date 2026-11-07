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
	first, second := 0, 0

	err := pullintake.AdmissionOfEach(
		countedAdmission{awaits: &first}, nil, countedAdmission{awaits: &second},
	).AwaitAdmission(context.Background())

	if err != nil || first != 1 || second != 1 {
		t.Errorf("err = %v, awaits = %d and %d, want both awaited once", err, first, second)
	}
}

func TestAdmissionOfEachStopsAtTheFirstAdmissionThatEnds(t *testing.T) {
	first, second := 0, 0

	err := pullintake.AdmissionOfEach(
		countedAdmission{awaits: &first, err: errHeldBack}, countedAdmission{awaits: &second},
	).AwaitAdmission(context.Background())

	if !errors.Is(err, errHeldBack) || second != 0 {
		t.Errorf("err = %v, second awaited %d times, want the first failure only", err, second)
	}
}
