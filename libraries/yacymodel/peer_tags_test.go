package yacymodel_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestParseTag(t *testing.T) {
	tag, err := yacymodel.ParseTag("search")
	if err != nil || tag.String() != "search" {
		t.Fatalf("ParseTag = %q, %v", tag.String(), err)
	}
}

func TestParseTagRejects(t *testing.T) {
	tooLong := string(make([]byte, yacymodel.MaxTagLength+1))
	for _, s := range []string{"", "a=b", "a|b", "a,b", "{x}", tooLong} {
		if _, err := yacymodel.ParseTag(s); !errors.Is(err, yacymodel.ErrBadTag) {
			t.Fatalf("ParseTag(%q) = %v, want ErrBadTag", s, err)
		}
	}
}

func TestMatchAllTags(t *testing.T) {
	if !yacymodel.MatchAllTags().MatchesAll() {
		t.Fatal("MatchAllTags does not match all")
	}
}

func TestNewPeerTags(t *testing.T) {
	tag, _ := yacymodel.ParseTag("news")
	tags, err := yacymodel.NewPeerTags([]yacymodel.Tag{tag})
	if err != nil || tags.MatchesAll() || len(tags.Tags()) != 1 {
		t.Fatalf("NewPeerTags = %v, %v", tags, err)
	}
}

func TestNewPeerTagsRejectsOverflow(t *testing.T) {
	many := make([]yacymodel.Tag, yacymodel.MaxTagsPerPeer+1)
	if _, err := yacymodel.NewPeerTags(many); !errors.Is(err, yacymodel.ErrBadPeerTags) {
		t.Fatalf("NewPeerTags overflow = %v, want ErrBadPeerTags", err)
	}
}
