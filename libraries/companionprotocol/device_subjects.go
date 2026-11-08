package companionprotocol

import "strings"

const (
	deviceSubjectStart = "companions."
	deviceInboxStart   = "_INBOX."
	settingsEnd        = ".settings"
	appEnd             = ".app"
	unpairEnd          = ".unpair"

	SettingsSubjects = deviceSubjectStart + "*" + settingsEnd
	AppSubjects      = deviceSubjectStart + "*" + appEnd
	UnpairSubjects   = deviceSubjectStart + "*" + unpairEnd
)

func SettingsSubjectOf(device DeviceID) string {
	return deviceSubjectStart + string(device) + settingsEnd
}

func AppSubjectOf(device DeviceID) string {
	return deviceSubjectStart + string(device) + appEnd
}

func UnpairSubjectOf(device DeviceID) string {
	return deviceSubjectStart + string(device) + unpairEnd
}

func DeviceSubjectsOf(device DeviceID) string {
	return deviceSubjectStart + string(device) + ".>"
}

func DeviceInboxPrefixOf(device DeviceID) string {
	return deviceInboxStart + string(device)
}

func DeviceRepliesOf(device DeviceID) string {
	return DeviceInboxPrefixOf(device) + ".>"
}

func DeviceFromSubject(subject string) DeviceID {
	device, _, _ := strings.Cut(strings.TrimPrefix(subject, deviceSubjectStart), ".")
	return DeviceID(device)
}
