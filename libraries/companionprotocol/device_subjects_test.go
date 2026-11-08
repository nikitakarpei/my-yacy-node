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
		if got := companionprotocol.DeviceIDFrom(subject); got != "device-a" {
			t.Errorf("device of %q = %q, want device-a", subject, got)
		}
	}
}

func TestTheInboxOfADeviceIsItsOwn(t *testing.T) {
	if companionprotocol.DeviceInboxPrefixOf("a") == companionprotocol.DeviceInboxPrefixOf("b") {
		t.Error("two devices share an inbox")
	}
	if companionprotocol.DeviceInboxSubjectsOf(
		"a",
	) != companionprotocol.DeviceInboxPrefixOf(
		"a",
	)+".>" {
		t.Errorf("inbox subjects = %q", companionprotocol.DeviceInboxSubjectsOf("a"))
	}
	if companionprotocol.DeviceSubjectsOf("a") != "devices.a.>" {
		t.Errorf("device subjects = %q", companionprotocol.DeviceSubjectsOf("a"))
	}
}

func TestASubjectOutsideTheDeviceSubjectsNamesNoDevice(t *testing.T) {
	for _, subject := range []string{"pairing.request", "devices.only-id", ""} {
		if got := companionprotocol.DeviceIDFrom(subject); got != "" {
			t.Errorf("device of %q = %q, want none", subject, got)
		}
	}
}
