//go:build e2e

package e2e

import "github.com/nikitakarpei/yacy-rwi-node/e2eharness/nodepeer"

var (
	distributionNodeKey     = nodepeer.KeyAfterFleet(0)
	postingAdmissionNodeKey = nodepeer.KeyAfterFleet(1)
	searchNodeKey           = nodepeer.KeyAfterFleet(2)
	promotionNodeKey        = nodepeer.KeyAfterFleet(3)
	verifyingNodeKey        = nodepeer.KeyAfterFleet(4)
	boundNodeKey            = nodepeer.KeyAfterFleet(5)
	callingNodeKey          = nodepeer.KeyAfterFleet(6)
	secureNodeKey           = nodepeer.KeyAfterFleet(7)
	closedSecurePortNodeKey = nodepeer.KeyAfterFleet(8)
)
