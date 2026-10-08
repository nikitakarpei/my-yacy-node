//go:build e2e

package yacypeer

func DHTDistributionOverrides() []string {
	const init = "/opt/yacy_search_server/defaults/yacy.init"

	return []string{
		"sed -i 's#^wordCacheMaxCount.*#wordCacheMaxCount = 1#' " + init,
	}
}
