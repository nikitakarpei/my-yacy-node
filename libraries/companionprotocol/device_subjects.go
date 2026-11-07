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

func SettingsSubjectOf(device string) string {
	return deviceSubjectStart + device + settingsEnd
}

func AppSubjectOf(device string) string {
	return deviceSubjectStart + device + appEnd
}

func UnpairSubjectOf(device string) string {
	return deviceSubjectStart + device + unpairEnd
}

func DeviceSubjectsOf(device string) string {
	return deviceSubjectStart + device + ".>"
}

func DeviceInboxPrefixOf(device string) string {
	return deviceInboxStart + device
}

func DeviceRepliesOf(device string) string {
	return DeviceInboxPrefixOf(device) + ".>"
}

func DeviceFromSubject(subject string) string {
	device, _, _ := strings.Cut(strings.TrimPrefix(subject, deviceSubjectStart), ".")
	return device
}
