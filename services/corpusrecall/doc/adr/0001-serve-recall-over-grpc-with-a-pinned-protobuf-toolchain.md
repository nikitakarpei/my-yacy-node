# 1. Serve recall over gRPC with a pinned protobuf toolchain

Status: accepted

## Context

`corpusrecall` answers on-demand requests for an operator's own corpus
representation of a URL. Callers are other services and edge adapters, not
browsers, and each request names a URL and the representation kinds it wants and
receives a structured reply carrying a heterogeneous set of representations
(markdown now, text later) plus the kinds that stayed unavailable.

The reply is a closed set of typed shapes, not free-form bytes, and the request is
a small typed record. A schema-first contract with generated stubs on both sides
gives callers a typed client, rejects malformed requests at the edge, and lets a
new representation kind be added as a new message without breaking existing
callers. The node has no other synchronous request/response API yet, so this is
the first such contract in the repository.

gRPC with protocol buffers is the established fit for typed service-to-service
calls and streaming-ready evolution. A JSON/HTTP API over hand-written types is
the alternative; it needs no code generation but gives no schema, no generated
clients, and no wire-level compatibility discipline.

## Decision

Define the recall contract as protocol buffers in `libraries/corpusrecallapi` and
serve it over gRPC. Depend on `google.golang.org/grpc` and
`google.golang.org/protobuf` at runtime.

Generate the Go stubs with `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc`,
each pinned in the build toolchain and run by `make proto`. Check the generated
`*.pb.go` into the repository so building the service needs no generator.

## Consequences

- Callers get a generated, typed client and server; unknown representation kinds
  are rejected at the gRPC edge before reaching retrieval.
- A new representation kind is a new message in the schema and a new source behind
  the retrieval port, not a breaking change to the wire contract.
- The repository carries generated code; `make proto` regenerates it
  reproducibly and CI fails if the checked-in stubs drift from the schema.
- The node gains gRPC and protobuf as runtime dependencies and a protobuf code
  generator as a build dependency, all version-pinned.
- A future MCP or firecrawl edge adapter is a separate front end over this same
  gRPC port, not a second contract in this service.
