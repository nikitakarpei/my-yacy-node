Code structure: Favor extension over modification. New capability → new unit (file or package) on the smallest seam; behavior change → edit in place. Split a unit when responsibilities diverge; collapse a seam when it couples units tightly.

Logging: Nothing fails silently. Happy paths: DEBUG. Recoverable failures: WARN. Failures that need an operator to intervene: ERROR.

Single source of truth: Every fact lives in exactly one place — a constant, config, or doc. Never restate it in comments, errors, logs, or a second doc.

Comments: No comments without explicit user approval. Use naming and structure instead. Godoc package docs are allowed.

Docs: Each doc is self-contained, plain-language, and at most 80 lines unless the user approves an exception. It covers only what someone using the system needs — no implementation detail, no rationale. Links are for navigation only. Write in neutral reference register: the system is the subject, not the reader. Name headers for the object or stage they describe, never for an activity. Every sentence must be true against the running system and earn its place; drop facts already implied and reassurances that ask nothing of the reader. State a security caveat as an instruction or omit it, never as a bare observation. Write every doc in ASD-STE100 Simplified Technical English: short sentences, one idea per sentence, active voice, and consistent, simple vocabulary.

Naming — responsibility: Every package, file, type, interface, port, function, method, field, and variable has one bounded responsibility, stateable in one sentence; if it cannot be, fix the abstraction. Prefer explicit bounded names over short generic ones. Never use `util.go`, `helpers.go`, `handler.go`, or `types.go`. Reject umbrella names such as Store, Manager, Service, Handler, Util, or catch-all domain names like Distribution*.

Naming — symmetry: When one variant is qualified, qualify every sibling the same way. Parallel implementations get parallel names (`elasticsearchSearchOnce`, `manticoreSearchOnce`); never leave one sibling bare.

Naming — style: Name a thing for what it is in the problem domain, not how it is built or what it is for. Strip implementation terms (count, map, hash, digest, buffer) and destination terms (shared, peer, abstract, response); keep the domain noun. Spell names in full; abbreviation is not allowed. Confine protocol- and transport-specific vocabulary to the edge that translates to and from it; inner code speaks plain domain language.

Dependencies: Record each new third-party dependency in its own ADR before use.

Version pinning: Pin all versions. Runtime deps: go.mod. Build/lint tools: Go tool directives in go.mod. make verify uses only pinned tools, never PATH versions.

Testing: Code lands with tests. make verify runs tests and coverage and is the only gate; a change is done only when it is green and total coverage stays at or above the threshold.

Coverage: If coverage drops, first remove or refactor code. Find uncovered statements/branches and evaluate whether they should exist: delete dead or defensive-only code, collapse unexercised branches, or replace several paths with one covered path. Add tests only for required behavior; filler tests written only to raise coverage fail the change.
