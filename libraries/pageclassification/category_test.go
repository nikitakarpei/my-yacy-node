package pageclassification_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

func TestALowercaseNameIsACategory(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"spam", "adult", "self-harm", "medical_2"} {
		category, err := pageclassification.CategoryFrom(name)
		if err != nil || string(category) != name {
			t.Errorf("%q gives %q and %v, want the category %q", name, category, err, name)
		}
	}
}

func TestANameThatAHeaderCannotCarryIsNoCategory(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"", "Spam", "2adult", "-adult", "adult content", "adult,spam"} {
		if _, err := pageclassification.CategoryFrom(name); err == nil {
			t.Errorf("%q is a category, want an error", name)
		}
	}
}
