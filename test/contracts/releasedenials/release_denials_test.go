package releasedenials_test

import (
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const releaseDenialsPath = "../../../distributions/yacypeercompanion/release-denials.json"

func TestTheReleaseDenialsOfTheCompanionAppAreReadable(t *testing.T) {
	content, err := os.ReadFile(releaseDenialsPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := trustedappreleases.DenialsFrom(content); err != nil {
		t.Errorf("the release denials are not readable: %v", err)
	}
}
