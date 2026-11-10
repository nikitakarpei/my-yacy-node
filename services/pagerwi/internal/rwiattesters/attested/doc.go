// Package attested signs a page-rwi statement over the page RWI root of each RWI, with a
// time-stamp, when the hardware key vouches for the page offer: it signed the page-offer
// statement, the statement covers the offer as received, the fetch URL is the page URL, the page
// URL is https on the default port, the landed URL is not too long, the drand round of the offer
// is of the pinned chain and is the fetch time, and the page has postings. Any other RWI goes
// without a page attestation.
package attested
