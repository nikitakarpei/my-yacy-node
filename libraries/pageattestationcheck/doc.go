// Package pageattestationcheck checks the page attestation of a URL metadata row in the
// Hardware-Attested Statements format. The row is attested when the attestation is a page-rwi
// statement over the page RWI root of the row, for the address of the row, with a time-stamp and
// a drand round, signed by a trusted app release. An attested row has the fetch time of its
// statement. A renewing check makes a new check on a timer from the current trusted releases.
package pageattestationcheck
