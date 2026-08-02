package replicashortfall

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func peersEligibleForReplicas(
	peers []yacymodel.Seed,
	eligibility ReplicaEligibility,
) []yacymodel.Seed {
	eligible := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if eligibility.Eligible(seed.Hash) {
			eligible = append(eligible, seed)
		}
	}

	return eligible
}
