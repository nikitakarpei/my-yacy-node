// Package pageattestationcheck checks the page attestation of a URL metadata row against a policy
// in the Hardware-Attested Statements format. The row is attested when the attestation is a
// page-rwi statement over the page RWI root of the row, for the address of the row, and verifies
// under the policy. An attested row has the fetch time of its statement. The check refuses a
// policy that does not require a time-stamp and a drand round. It keeps no state.
package pageattestationcheck
