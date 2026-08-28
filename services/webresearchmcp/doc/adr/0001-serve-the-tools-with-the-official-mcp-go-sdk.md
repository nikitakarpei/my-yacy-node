# 1. Serve the tools with the official MCP Go SDK

Status: accepted

## Context

`webresearchmcp` serves two tools to assistants over the Model Context Protocol. The
protocol is JSON-RPC 2.0 with a versioned handshake, a tool listing, tool calls, and a
streamable HTTP transport with session identifiers and resumable event streams. Every
part of that is protocol plumbing, and none of it is what this service is about.

`github.com/modelcontextprotocol/go-sdk` is the Go SDK published by the protocol's own
project, at v1.7.0, asking for Go 1.25. It carries the server, the tool registry,
argument schemas derived from Go types, and the stdio and streamable HTTP transports.
Its code is licensed Apache-2.0, except for contributions the project has not yet
relicensed from MIT.

`github.com/mark3labs/mcp-go` is the community alternative, at v0.58.0 with v1.0.0 in
beta, and it tracks the specification from outside the project that writes it.

Hand-rolling the protocol keeps the dependency list short, at the cost of owning the
handshake, the transport, and the version negotiation of a specification that changes
outside this repository.

## Decision

Depend on `github.com/modelcontextprotocol/go-sdk`, pinned to v1.7.0 in the service's
`go.mod`.

The dependency stays inside the package that serves the protocol endpoint. The
orchestrators behind it take no type from the SDK, so what the tools do is stated in
this repository's own vocabulary.

## Consequences

- The service follows the specification through its own project's releases, rather
  than through a reimplementation of it.
- A breaking change in the SDK reaches one package, and the pinned version keeps every
  upgrade on our own schedule.
- The service takes the SDK's own dependencies with it, among them a JSON Schema
  library, an OAuth2 library, and a JWT library.
- The SDK asks for Go 1.25, below the version this repository builds with, so it puts no
  floor of its own on the service.
- Replacing the SDK, or serving another protocol beside MCP, is a second package at
  the same seam.
