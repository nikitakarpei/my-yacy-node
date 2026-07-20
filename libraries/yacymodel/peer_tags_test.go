package yacymodel

import (
	"errors"
	"testing"
)

func TestParseTag(t *testing.T) {
	tag, err := ParseTag("search")
	if err != nil || tag.String() != "search" {
		t.Fatalf("ParseTag = %q, %v", tag.String(), err)
	}
}

func TestParseTagRejects(t *testing.T) {
	for _, s := range []string{"", "a=b", "a|b", "a,b", "{x}", string(make([]byte, 101))} {
		if _, err := ParseTag(s); !errors.Is(err, ErrBadTag) {
			t.Fatalf("ParseTag(%q) = %v, want ErrBadTag", s, err)
		}
	}
}

func TestMatchAllTags(t *testing.T) {
	if !MatchAllTags().MatchesAll() {
		t.Fatal("MatchAllTags does not match all")
	}
}

func TestNewPeerTags(t *testing.T) {
	tag, _ := ParseTag("news")
	tags, err := NewPeerTags([]Tag{tag})
	if err != nil || tags.MatchesAll() || len(tags.Tags()) != 1 {
		t.Fatalf("NewPeerTags = %v, %v", tags, err)
	}
}

func TestNewPeerTagsRejectsOverflow(t *testing.T) {
	many := make([]Tag, peerTagsMaxLen+1)
	if _, err := NewPeerTags(many); !errors.Is(err, ErrBadPeerTags) {
		t.Fatalf("NewPeerTags overflow = %v, want ErrBadPeerTags", err)
	}
}
