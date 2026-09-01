package yacyproto_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type failingSeedList struct {
	err error
}

func (r failingSeedList) Read([]byte) (int, error) {
	return 0, r.err
}

func TestParseSeedListResponseKeepsAddressableSeeds(t *testing.T) {
	t.Parallel()

	firstSeed := fullSeed(t)
	secondSeed := fullSeed(t)
	secondSeed.Hash = sampleHash(t, "second")
	secondSeed.Name = mustPeerName(t, "second-peer")
	seedList := strings.Join([]string{
		seedWireForm(firstSeed),
		"",
		"not-a-seed",
		seedWireForm(secondSeed),
	}, "\n")

	response, err := yacyproto.ParseSeedListResponse(t.Context(), strings.NewReader(seedList))
	if err != nil {
		t.Fatalf("parse seed list response: %v", err)
	}
	if len(response.Seeds) != 2 ||
		response.Seeds[0].Hash != firstSeed.Hash ||
		response.Seeds[1].Hash != secondSeed.Hash {
		t.Fatalf(
			"ParseSeedListResponse = %+v, want %s and %s",
			response,
			firstSeed.Hash,
			secondSeed.Hash,
		)
	}
}

func TestParseSeedListResponseDiscardsASeedWithoutAnAddress(t *testing.T) {
	t.Parallel()

	seed := sampleSeed(t, "alpha", "example-peer")
	response, err := yacyproto.ParseSeedListResponse(
		t.Context(),
		strings.NewReader(seedWireForm(seed)),
	)
	if err != nil {
		t.Fatalf("parse seed list response: %v", err)
	}
	if len(response.Seeds) != 0 {
		t.Fatalf("ParseSeedListResponse = %+v, want no unaddressable seed", response)
	}
}

func TestParseSeedListResponsePassesAReadFailure(t *testing.T) {
	t.Parallel()

	want := errors.New("read failed")
	_, err := yacyproto.ParseSeedListResponse(t.Context(), failingSeedList{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("ParseSeedListResponse error = %v, want %v", err, want)
	}
}
