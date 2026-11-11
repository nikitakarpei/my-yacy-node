package pageattestationchecktest

import (
	"encoding/json"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
)

type AcceptedVersion struct {
	VersionCode int64
	EndTime     time.Time
}

func PolicyFor(authority *TimeStampAuthority, versions ...AcceptedVersion) []byte {
	versionCodes := []int64{}
	versionEndTimes := []map[string]int64{}
	for _, version := range versions {
		versionCodes = append(versionCodes, version.VersionCode)
		if !version.EndTime.IsZero() {
			versionEndTimes = append(versionEndTimes, map[string]int64{
				"versionCode": version.VersionCode,
				"endTime":     version.EndTime.Unix(),
			})
		}
	}
	signer := map[string]any{
		"profile":             android.Name,
		"package":             AppPackage,
		"signingCertificates": []string{AppSigningCertificate},
		"versionCodes":        versionCodes,
	}
	if len(versionEndTimes) > 0 {
		signer["versionEndTimes"] = versionEndTimes
	}
	policy, _ := json.Marshal(map[string]any{ //nolint:errchkjson // strings, numbers and raw JSON.
		"policy":               verifier.PolicyVersion,
		"timestampAuthorities": []json.RawMessage{authority.policyEntry()},
		"drandChains":          []string{"quicknet"},
		"predicateTypes": map[string]any{
			pagerwicontract.PageRWIPredicateType: map[string]any{
				"trustedSigners":      []map[string]any{signer},
				"requireTimestamp":    true,
				"requireStartedAfter": true,
			},
		},
	})

	return policy
}
