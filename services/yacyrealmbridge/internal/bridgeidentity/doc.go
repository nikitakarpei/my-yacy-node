// Package bridgeidentity owns the bridge's own peer in one realm: the seed it
// publishes there, marked as a bridge seed and offering no capability, and the
// peer endpoints that seed promises. Its greeting hands the caller to the view
// of that realm and answers with the peers the bridge holds in the other realm,
// at the addresses they are translated to.
package bridgeidentity
