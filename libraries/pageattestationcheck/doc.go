// Package pageattestationcheck checks the page attestation of a URL metadata row in the
// Hardware-Attested Statements format. The row is attested when the attestation is a page-rwi
// statement over the page RWI root of the row, for the address of the row, with a time-stamp and
// a drand round, signed by a trusted app release. A policy names the time-stamp authorities and
// drand chains. An attested row has the fetch time of its statement. The check keeps no state.
package pageattestationcheck
