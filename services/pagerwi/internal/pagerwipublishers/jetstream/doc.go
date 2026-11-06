// Package jetstream publishes the RWI of each page the service builds on the RWI stream, which
// holds it until every node consumer has taken it. How each publication fared goes to the
// observer; the caller learns only whether the stream took the RWI.
package jetstream
