package companionprotocol_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachSubjectOfADeviceNamesTheDevice(t *testing.T) {
	for _, subject := range []string{
		companionprotocol.HelloSubjectOf("device-a"),
		companionprotocol.SettingsSubjectOf("device-a"),
		companionprotocol.UnpairingRequestSubjectOf("device-a"),
	} {
		if got := companionprotocol.DeviceIDIn(subject); got != "device-a" {
			t.Errorf("device id in %q = %q, want device-a", subject, got)
		}
	}
}

func TestEachFilterCoversItsOwnKindOfSubjectForEveryDevice(t *testing.T) {
	subjectOfEachFilter := map[string]func(companionprotocol.DeviceID) string{
		companionprotocol.HelloSubjects:            companionprotocol.HelloSubjectOf,
		companionprotocol.SettingsSubjects:         companionprotocol.SettingsSubjectOf,
		companionprotocol.UnpairingRequestSubjects: companionprotocol.UnpairingRequestSubjectOf,
	}
	for filter, subjectOf := range subjectOfEachFilter {
		for otherFilter, otherSubjectOf := range subjectOfEachFilter {
			subject := otherSubjectOf("device-a")
			if covers(filter, subject) != (filter == otherFilter) {
				t.Errorf("%q covers %q = %v", filter, subject, covers(filter, subject))
			}
		}
		if !covers(filter, subjectOf("device-b")) {
			t.Errorf("%q does not cover the subject of device-b", filter)
		}
	}
}

func TestTheDeviceSubjectsCoverEachSubjectOfTheDeviceOnly(t *testing.T) {
	for _, subject := range []string{
		companionprotocol.HelloSubjectOf("device-a"),
		companionprotocol.SettingsSubjectOf("device-a"),
		companionprotocol.UnpairingRequestSubjectOf("device-a"),
	} {
		if !covers(companionprotocol.DeviceSubjectsOf("device-a"), subject) {
			t.Errorf("the subjects of device-a do not cover %q", subject)
		}
		if covers(companionprotocol.DeviceSubjectsOf("device-b"), subject) {
			t.Errorf("the subjects of device-b cover %q", subject)
		}
	}
}

func TestTheInboxSubjectsOfADeviceCoverItsInboxesOnly(t *testing.T) {
	inbox := companionprotocol.DeviceInboxPrefixOf("device-a") + ".reply-1"

	if !covers(companionprotocol.DeviceInboxSubjectsOf("device-a"), inbox) {
		t.Errorf("the inbox subjects of device-a do not cover %q", inbox)
	}
	if covers(companionprotocol.DeviceInboxSubjectsOf("device-b"), inbox) {
		t.Errorf("the inbox subjects of device-b cover %q", inbox)
	}
}

func covers(filter string, subject string) bool {
	filterTokens := strings.Split(filter, ".")
	subjectTokens := strings.Split(subject, ".")
	for index, filterToken := range filterTokens {
		if filterToken == ">" {
			return len(subjectTokens) > index
		}
		if index >= len(subjectTokens) ||
			filterToken != "*" && filterToken != subjectTokens[index] {
			return false
		}
	}
	return len(filterTokens) == len(subjectTokens)
}
