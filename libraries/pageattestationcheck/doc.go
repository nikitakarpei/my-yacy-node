// Package pageattestationcheck checks the page attestation of a URL metadata row in the
// Hardware-Attested Statements format. The row is attested when the attestation is a page-rwi
// statement over the page RWI root of the row, for the address of the row, with a time-stamp of
// the companion app's authority and a drand round, signed by a trusted app release. An attested
// row has the fetch time of its statement. The check renews its trusted releases each hour.
package pageattestationcheck
