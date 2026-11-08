package companionprotocol

import "strings"

const (
	deviceSubjectPrefix    = "devices."
	deviceInboxesPrefix    = "_INBOX."
	settingsSuffix         = ".settings"
	appReportSuffix        = ".app-report"
	unpairingRequestSuffix = ".unpairing-request"

	SettingsSubjects         = deviceSubjectPrefix + "*" + settingsSuffix
	AppReportSubjects        = deviceSubjectPrefix + "*" + appReportSuffix
	UnpairingRequestSubjects = deviceSubjectPrefix + "*" + unpairingRequestSuffix
)

func SettingsSubjectOf(deviceID DeviceID) string {
	return deviceSubjectPrefix + string(deviceID) + settingsSuffix
}

func AppReportSubjectOf(deviceID DeviceID) string {
	return deviceSubjectPrefix + string(deviceID) + appReportSuffix
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
