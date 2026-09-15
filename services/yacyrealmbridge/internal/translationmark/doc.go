// Package translationmark owns the tags a public bridge puts on every seed it
// translates, stating when the bridge first leased that address for that hash,
// signed with the bridge's key over the seed they mark. It tells whether a seed
// carries a translation mark at all, and what a mark signed under one of the
// configured trusted keys states about the seed.
package translationmark
