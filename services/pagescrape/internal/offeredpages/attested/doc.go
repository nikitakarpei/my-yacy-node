// Package attested offers each fetched page with a page-offer statement that the hardware key
// signs. It signs only when that key signed the capture of each hop, each capture names the URL
// that the hop asked for, and the last capture names the offered body and a drand round of the
// pinned chain. The page is then fetched at the time of that round. Any other page is offered
// without an attestation, as fetched at the time of the clock.
package attested
