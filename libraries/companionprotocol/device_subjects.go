package companionprotocol

import "strings"

const (
	deviceSubjectPrefix = "devices."
	deviceInboxPrefix   = "_INBOX."
	settingsSuffix      = ".settings"
	appSuffix           = ".app"
	unpairSuffix        = ".unpair"

	SettingsSubjects = deviceSubjectPrefix + "*" + settingsSuffix
	AppSubjects      = deviceSubjectPrefix + "*" + appSuffix
	UnpairSubjects   = deviceSubjectPrefix + "*" + unpairSuffix
)

func SettingsSubjectOf(device DeviceID) string {
	return deviceSubjectPrefix + string(device) + settingsSuffix
}

func AppSubjectOf(device DeviceID) string {
	return deviceSubjectPrefix + string(device) + appSuffix
}

func UnpairSubjectOf(device DeviceID) string {
	return deviceSubjectPrefix + string(device) + unpairSuffix
}

func DeviceSubjectsOf(device DeviceID) string {
	return deviceSubjectPrefix + string(device) + ".>"
}

func DeviceInboxPrefixOf(device DeviceID) string {
	return deviceInboxPrefix + string(device)
}

func DeviceInboxSubjectsOf(device DeviceID) string {
	return DeviceInboxPrefixOf(device) + ".>"
}

func DeviceIDFrom(subject string) DeviceID {
	rest, deviceSubject := strings.CutPrefix(subject, deviceSubjectPrefix)
	device, _, found := strings.Cut(rest, ".")
	if !deviceSubject || !found {
		return ""
	}
	return DeviceID(device)
}
