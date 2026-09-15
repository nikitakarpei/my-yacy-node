// Package realmpeerview owns what the bridge holds of one realm: the seed of
// every peer it holds there by hash, and the addresses the trusted translations
// of that realm carry. It takes in every seed the realm offers, holds a peer
// only once a YaCy peer of the network answers at the address the seed
// advertises, asks again on a schedule, and drops a peer that stops answering.
// Every instance of one bridge holds the same view of a realm.
package realmpeerview
