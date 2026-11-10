// Package pageattestationcheck checks the page attestation of a URL metadata row. The row is
// attested when the attestation is a page-rwi statement over the page RWI root of the row, for the
// address of the row, signed by an app version that the policy accepts at the time of the
// statement, and the statement verifies as a Hardware-Attested Statement under the policy. An
// attested row has the fetch time of its statement. The check keeps no state.
package pageattestationcheck
