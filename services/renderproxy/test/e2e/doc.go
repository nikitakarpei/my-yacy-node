// Package e2e runs the end-to-end renderproxy test against containers.
//
// It starts a scripted static origin, a lightpanda CDP browser, an egress
// proxy, and the renderproxy image on hermetic Docker networks, then drives
// renderproxy from the host as a forward proxy and reads back what it answers.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e`. It is not part of the `make verify` quality gate.
package e2e
