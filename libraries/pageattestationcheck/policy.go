package pageattestationcheck

import (
	"encoding/json"
	"time"
)

type Policy struct {
	Package              string
	SigningCertificates  []string
	AcceptedVersions     []AcceptedVersion
	TimestampAuthorities []json.RawMessage
}

type AcceptedVersion struct {
	VersionCode int64
	EndTime     time.Time
}

var FreeTSA = json.RawMessage(`"freetsa"`)
