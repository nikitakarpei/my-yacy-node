package companionprotocol

import "strings"

const (
	deviceSubjectPrefix    = "devices."
	deviceInboxesPrefix    = "_INBOX."
	helloSuffix            = ".hello"
	settingsSuffix         = ".settings"
	unpairingRequestSuffix = ".unpairing-request"

	HelloSubjects            = deviceSubjectPrefix + "*" + helloSuffix
	SettingsSubjects         = deviceSubjectPrefix + "*" + settingsSuffix
	UnpairingRequestSubjects = deviceSubjectPrefix + "*" + unpairingRequestSuffix
)

func HelloSubjectOf(deviceID DeviceID) string {
	return deviceSubjectPrefix + string(deviceID) + helloSuffix
}

func SettingsSubjectOf(deviceID DeviceID) string {
	return deviceSubjectPrefix + string(deviceID) + settingsSuffix
}

func UnpairingRequestSubjectOf(deviceID DeviceID) string {
	return deviceSubjectPrefix + string(deviceID) + unpairingRequestSuffix
}

func DeviceSubjectsOf(deviceID DeviceID) string {
	return deviceSubjectPrefix + string(deviceID) + ".>"
}

func DeviceInboxPrefixOf(deviceID DeviceID) string {
	return deviceInboxesPrefix + string(deviceID)
}

func DeviceInboxSubjectsOf(deviceID DeviceID) string {
	return DeviceInboxPrefixOf(deviceID) + ".>"
}

func DeviceIDIn(subject string) DeviceID {
	deviceID, _, _ := strings.Cut(strings.TrimPrefix(subject, deviceSubjectPrefix), ".")
	return DeviceID(deviceID)
}
