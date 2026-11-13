package pageattestationchecktest

import (
	"encoding/json"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
)

func PolicyFor(authority *TimeStampAuthority) []byte {
	policy, _ := json.Marshal(map[string]any{ //nolint:errchkjson // strings and raw JSON.
		"policy":               verifier.PolicyVersion,
		"timestampAuthorities": []json.RawMessage{authority.policyEntry()},
		"drandChains":          []string{"quicknet"},
	})

	return policy
}
