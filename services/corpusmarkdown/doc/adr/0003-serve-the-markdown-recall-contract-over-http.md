# 3. Serve the markdown recall contract over HTTP

Status: accepted

## Context

`corpusmarkdown` served one call, `RecallPage`, over gRPC. `webresearchmcp` was its only
caller. Both are Go services in this repository, and no other reader spoke the contract.

That one call cost the repository two runtime dependencies, `google.golang.org/grpc` and
`google.golang.org/protobuf`, in three modules. It also cost three pinned build tools,
`protoc` with its bundled type definitions, `protoc-gen-go`, and `protoc-gen-go-grpc`, a
`make proto` step, and 335 lines of generated code under version control.

No architecture decision recorded the gRPC dependency, which the repository requires
before use.

Every other endpoint in the stack is HTTP with JSON, and `corpusmarkdown` already runs an
HTTP server for `/metrics`. The call is one request and one answer, with no stream.

## Decision

The corpus serves the recall contract as `GET /page/markdown`, with the requested URL in
the `url` query parameter, on the same address as before. The answer is a JSON body, and
the contract module declares the path, the parameter, and that body as Go types.

The corpus answers 404 for a URL it holds no markdown for, 400 for a URL it cannot make
canonical, and 500 for a corpus it cannot read.

The recall server joins the ops server in the service's server group, which shuts both
down together.

## Consequences

- The repository drops both gRPC dependencies and all three protobuf build tools.
- The contract module depends on no third party.
- Operators change nothing: the address, the port, and the variables that name them stay.
- A future non-Go reader gets no generated stub, and writes an HTTP call by hand.
- A future streaming answer needs a new decision, not an added method.
