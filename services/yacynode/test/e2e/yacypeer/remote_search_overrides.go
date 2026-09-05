//go:build e2e

package yacypeer

// RemoteSearchOverrides relaxes the two YaCy settings that make a remote RWI
// search skip an otherwise eligible peer: the system load ceiling
// (RemoteSearch.java:292) and the per-peer time budget, which the default
// 3000ms leaves too tight for a cold container.
func RemoteSearchOverrides() []string {
	const init = "/opt/yacy_search_server/defaults/yacy.init"

	return []string{
		"sed -i 's#^remotesearch.maxload.rwi.*#remotesearch.maxload.rwi=999.0#' " + init,
		"sed -i 's#^remotesearch.maxtime.*#remotesearch.maxtime = 10000#' " + init,
	}
}
