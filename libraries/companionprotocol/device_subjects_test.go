package companionprotocol_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachSubjectOfADeviceNamesTheDevice(t *testing.T) {
	for _, subject := range []string{
		companionprotocol.SettingsSubjectOf("device-a"),
		companionprotocol.AppSubjectOf("device-a"),
		companionprotocol.UnpairSubjectOf("device-a"),
	} {
		if got := companionprotocol.DeviceFromSubject(subject); got != "device-a" {
			t.Errorf("device of %q = %q, want device-a", subject, got)
		}
	}
}

func TestTheInboxOfADeviceIsItsOwn(t *testing.T) {
	if companionprotocol.DeviceInboxPrefixOf("a") == companionprotocol.DeviceInboxPrefixOf("b") {
		t.Error("two devices share an inbox")
	}
	if companionprotocol.DeviceRepliesOf("a") != companionprotocol.DeviceInboxPrefixOf("a")+".>" {
		t.Errorf("replies = %q", companionprotocol.DeviceRepliesOf("a"))
	}
	if companionprotocol.DeviceSubjectsOf("a") != "companions.a.>" {
		t.Errorf("device subjects = %q", companionprotocol.DeviceSubjectsOf("a"))
	}
}
