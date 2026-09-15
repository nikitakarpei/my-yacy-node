// Package realmpeerview owns the bridge's view of one realm: the seed of every
// peer it confirmed there by hash, and the addresses the trusted translations
// of that realm carry. It takes in every seed the realm offers, confirms a peer
// only once a YaCy peer of the network answers at the address the seed
// advertises, asks again on a schedule, and drops a peer that stops answering.
// Every instance of one bridge has the same view of a realm.
package realmpeerview
