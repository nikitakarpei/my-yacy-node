// Package peerrequestforwarding serves the YaCy peer-protocol paths on every
// translated address of a realm. Each path resolves the address the request
// arrived at to the confirmed peer behind it, reads what the operator
// admits on that path for that crossing, and either answers the refusal in
// place of that peer, in the vocabulary of the path, or forwards the request
// with every seed it carries translated for the realm it enters.
package peerrequestforwarding
