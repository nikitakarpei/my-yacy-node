package vault_test

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type sighting struct {
	word   string
	seenAt time.Time
}

func sightingOf(word, seconds string) sighting {
	unixSeconds, _ := strconv.ParseInt(seconds, 10, 64)

	return sighting{word: word, seenAt: time.Unix(unixSeconds, 0).UTC()}
}

var firstSightingAt = time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC)

func secondsAfterFirstSighting(seconds int) time.Time {
	return firstSightingAt.Add(time.Duration(seconds) * time.Second)
}

type sightings struct {
	vault     *vault.Vault
	lastSeen  *vault.Collection[string, string]
	retention *vault.RetentionIndex[sighting, string]
}

func openSightings(t *testing.T) sightings {
	t.Helper()

	v, err := openDouble()
	if err != nil {
		t.Fatalf("openDouble: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	retention, err := v.RegisterRetentionIndex(
		vault.Name("sighting_retention"),
		firstVersion,
		stringKeyLayout,
		func(seen sighting) time.Time { return seen.seenAt },
		func(seen sighting) string { return seen.word },
	)
	if err != nil {
		t.Fatalf("RegisterRetentionIndex: %v", err)
	}
	lastSeen, err := v.RegisterProjectedCollection(
		context.Background(),
		vault.Name("last_seen"),
		stringKeyLayout,
		stringValueCodec{},
		vault.ProjectedAs(sightingOf, retention),
	)
	if err != nil {
		t.Fatalf("RegisterProjectedCollection: %v", err)
	}

	return sightings{vault: v, lastSeen: lastSeen, retention: retention}
}

func (s sightings) see(t *testing.T, word string, seenAt time.Time) {
	t.Helper()

	s.write(t, func(tx *vault.Txn) error {
		_, err := s.lastSeen.Put(tx, word, strconv.FormatInt(seenAt.Unix(), 10))

		return err
	})
}

func (s sightings) forget(t *testing.T, word string) {
	t.Helper()

	s.write(t, func(tx *vault.Txn) error {
		_, err := s.lastSeen.Delete(tx, word)

		return err
	})
}

func (s sightings) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := s.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (s sightings) wordsSeenBefore(t *testing.T, cutoff time.Time, limit int) []string {
	t.Helper()

	var words []string
	if err := s.vault.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		words, err = s.retention.KeysBefore(tx, cutoff, limit)

		return err
	}); err != nil {
		t.Fatalf("View: %v", err)
	}

	return words
}

func TestARetentionIndexGivesTheKeysRetainedBeforeTheCutoffOldestFirst(t *testing.T) {
	s := openSightings(t)
	s.see(t, "a", secondsAfterFirstSighting(2))
	s.see(t, "b", secondsAfterFirstSighting(0))
	s.see(t, "c", secondsAfterFirstSighting(1))
	s.see(t, "d", secondsAfterFirstSighting(5))

	got := s.wordsSeenBefore(t, secondsAfterFirstSighting(5), 10)

	if want := []string{"b", "c", "a"}; !slices.Equal(got, want) {
		t.Fatalf("KeysBefore = %v, want %v", got, want)
	}
}

func TestARetentionIndexGivesNoMoreKeysThanTheLimit(t *testing.T) {
	s := openSightings(t)
	s.see(t, "a", secondsAfterFirstSighting(0))
	s.see(t, "b", secondsAfterFirstSighting(1))
	s.see(t, "c", secondsAfterFirstSighting(2))

	if got, want := s.wordsSeenBefore(
		t,
		secondsAfterFirstSighting(5),
		2,
	), []string{
		"a",
		"b",
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("KeysBefore = %v, want %v", got, want)
	}
	if got := s.wordsSeenBefore(t, secondsAfterFirstSighting(5), 0); len(got) != 0 {
		t.Fatalf("KeysBefore with no room = %v, want none", got)
	}
}

func TestARewrittenRecordIsRetainedFromItsLatestTime(t *testing.T) {
	s := openSightings(t)
	s.see(t, "a", secondsAfterFirstSighting(0))
	s.see(t, "a", secondsAfterFirstSighting(10))

	if got := s.wordsSeenBefore(t, secondsAfterFirstSighting(5), 10); len(got) != 0 {
		t.Fatalf("KeysBefore = %v, want none", got)
	}
	if got, want := s.wordsSeenBefore(
		t,
		secondsAfterFirstSighting(11),
		10,
	), []string{
		"a",
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("KeysBefore = %v, want %v", got, want)
	}
}

func TestADeletedRecordLeavesTheRetentionIndex(t *testing.T) {
	s := openSightings(t)
	s.see(t, "a", secondsAfterFirstSighting(0))
	s.see(t, "b", secondsAfterFirstSighting(1))
	s.forget(t, "a")

	if got, want := s.wordsSeenBefore(
		t,
		secondsAfterFirstSighting(5),
		10,
	), []string{
		"b",
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("KeysBefore = %v, want %v", got, want)
	}
}
