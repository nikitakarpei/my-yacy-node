// Package attested signs a page-rwi statement over the page RWI root of each RWI, with a
// time-stamp, when the hardware key vouches for the page offer: it signed the page-offer
// statement, the offer equals that statement, the fetch URL is the page URL, the page URL is
// https on the default port, the landed URL is not too long, the drand round of the offer
// verifies and is the fetch time, and the page has postings. Any other RWI goes without a page
// attestation.
package attested
