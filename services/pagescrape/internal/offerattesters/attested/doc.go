// Package attested dates each fetched page by the drand round in its response attestations and
// signs a page-offer statement for it with the hardware key. It does so only when that key signed
// the response attestation of each hop, each names the URL that the hop asked for, and the last
// names the offered body and a round of the pinned chain. Any other page is dated by the clock and
// goes without an attestation.
package attested
