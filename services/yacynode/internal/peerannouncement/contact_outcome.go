package peerannouncement

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ContactOutcome string

const (
	PeerDidNotAnswer       ContactOutcome = "did_not_answer"
	PeerReportedNoSelfType ContactOutcome = "reported_no_self_type"
)

func ContactOutcomeFromReportedSelfType(reportedSelfType yacymodel.PeerType) ContactOutcome {
	return ContactOutcome("reported_self_" + reportedSelfType.String())
}

func contactOutcomeFromObservedPeerType(
	observedPeerType yacymodel.Optional[yacymodel.PeerType],
) ContactOutcome {
	reportedSelfType, present := observedPeerType.Get()
	if !present {
		return PeerReportedNoSelfType
	}

	return ContactOutcomeFromReportedSelfType(reportedSelfType)
}

func ContactOutcomes() []ContactOutcome {
	reportedSelfTypes := yacymodel.PeerTypes()
	outcomes := make([]ContactOutcome, 0, len(reportedSelfTypes)+2)
	outcomes = append(outcomes, PeerDidNotAnswer, PeerReportedNoSelfType)
	for _, reportedSelfType := range reportedSelfTypes {
		outcomes = append(outcomes, ContactOutcomeFromReportedSelfType(reportedSelfType))
	}

	return outcomes
}

func amountOfPeersPerContactOutcome(contactOutcomes []ContactOutcome) map[ContactOutcome]int {
	amounts := make(map[ContactOutcome]int, len(ContactOutcomes()))
	for _, outcome := range ContactOutcomes() {
		amounts[outcome] = 0
	}
	for _, outcome := range contactOutcomes {
		amounts[outcome]++
	}

	return amounts
}
