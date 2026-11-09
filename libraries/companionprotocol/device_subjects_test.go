package companionprotocol_test

import (
	"crypto/x509"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestEachSubjectOfADeviceKeepsItsWireForm(t *testing.T) {
	for _, subject := range []struct{ got, want string }{
		{companionprotocol.HelloSubjectOf("d"), "devices.d.hello"},
		{companionprotocol.SettingsSubjectOf("d"), "devices.d.settings"},
		{companionprotocol.UnpairingRequestSubjectOf("d"), "devices.d.unpairing-request"},
		{companionprotocol.DeviceInboxPrefixOf("d"), "_INBOX.d"},
	} {
		if subject.got != subject.want {
			t.Errorf("subject = %q, want %q", subject.got, subject.want)
		}
	}
}

func TestEachSubjectOfADeviceNamesTheDeviceOfItsKey(t *testing.T) {
	deviceID := deviceIDOfARandomKey(t)
	for _, subject := range []string{
		companionprotocol.HelloSubjectOf(deviceID),
		companionprotocol.SettingsSubjectOf(deviceID),
		companionprotocol.UnpairingRequestSubjectOf(deviceID),
	} {
		if got := companionprotocol.DeviceIDIn(subject); got != deviceID {
			t.Errorf("device id in %q = %q, want %q", subject, got, deviceID)
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

func TestTheDeviceSubjectsCoverEachSubjectOfTheDeviceOfTheKeyOnly(t *testing.T) {
	deviceID := deviceIDOfARandomKey(t)
	otherDeviceID := deviceIDOfARandomKey(t)
	for _, subject := range []string{
		companionprotocol.HelloSubjectOf(deviceID),
		companionprotocol.SettingsSubjectOf(deviceID),
		companionprotocol.UnpairingRequestSubjectOf(deviceID),
	} {
		if !covers(companionprotocol.DeviceSubjectsOf(deviceID), subject) {
			t.Errorf("the subjects of the device do not cover %q", subject)
		}
		if covers(companionprotocol.DeviceSubjectsOf(otherDeviceID), subject) {
			t.Errorf("the subjects of another device cover %q", subject)
		}
	}
}

func TestTheInboxSubjectsOfADeviceCoverItsInboxesOnly(t *testing.T) {
	deviceID := deviceIDOfARandomKey(t)
	inbox := companionprotocol.DeviceInboxPrefixOf(deviceID) + ".reply-1"

	if !covers(companionprotocol.DeviceInboxSubjectsOf(deviceID), inbox) {
		t.Errorf("the inbox subjects of the device do not cover %q", inbox)
	}
	if covers(companionprotocol.DeviceInboxSubjectsOf(deviceIDOfARandomKey(t)), inbox) {
		t.Errorf("the inbox subjects of another device cover %q", inbox)
	}
}

func deviceIDOfARandomKey(t *testing.T) companionprotocol.DeviceID {
	t.Helper()
	return companionprotocol.DeviceIDFrom(randomPublicKeyInfo(t))
}

func randomPublicKeyInfo(t *testing.T) []byte {
	t.Helper()
	publicKeyInfo, err := x509.MarshalPKIXPublicKey(&randomDeviceKey(t).PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	return publicKeyInfo
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
