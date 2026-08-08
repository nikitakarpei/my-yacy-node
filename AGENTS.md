Code structure: New capability → new unit (file or package) on the smallest seam; behavior change → edit in place. Split a unit when responsibilities diverge; collapse a seam when it couples units tightly. Boundaries follow ownership, not control flow: a unit owns durable state, a protocol endpoint, or a value type and the rule computing it. A stage of a procedure is not ownable, and stages that only run in sequence are one unit. Exactly one unit per parent may instead own the whole pass.

Shared state: The service that owns a stream, bucket, or table is the only one that creates it. A contract package holds vocabulary — names, subjects, keys — never provisioning. A consumer opens what it reads and passes the broker's error through; it adds no precondition check. Which service provisions which state is a fact for the consumer's configuration doc, never for code, errors, or a contract package.

Reading order: A file is read once, top to bottom. The entry point comes first. Every function it names is defined below it in the order it is called, and each of those is followed the same way by the functions it names. A helper called twice in a file goes under its first caller.

Levels: A file holds one level of abstraction. A call chain deeper than one step is the signal that a middle level speaks its own vocabulary: give that level its own unit, a type in its own file or a package. A unit that would take every step of a stage is an umbrella; each step belongs to the unit that owns what it writes.

Cohesion: A type's methods and the private functions only it calls live in the file that declares it. A type is never split across files.

Abstraction level: A function orchestrates or it works, never both. An orchestrator only names steps and reads as the scenario in plain language. A working function does one thing and holds the detail its name hides. Log attributes, struct assembly, and error wrapping are detail; an orchestrator that holds them is split.

Logging: Nothing fails silently. Happy paths: DEBUG. Recoverable failures: WARN. Failures that need an operator to intervene: ERROR.

Single source of truth: Every fact lives in exactly one place — a constant, config, or doc. Never restate it in comments, errors, logs, or a second doc.

Comments: No comments without explicit user approval. Use naming and structure instead. Godoc package docs are allowed.

Docs: Each doc is self-contained and at most 80 lines unless the user approves an exception. Keep every paragraph to five lines or fewer. It covers only what someone using the system needs — no implementation detail, no rationale. Links are for navigation only. Write in neutral reference register: the system is the subject, not the reader. Name headers for the object or stage they describe, never for an activity. Every sentence must be true against the running system; drop facts already implied and reassurances that ask nothing of the reader. State a security caveat as an instruction or omit it. Write every doc in ASD-STE100 Simplified Technical English: short sentences, one idea per sentence, active voice, and consistent, simple vocabulary.

Naming — responsibility: Every package, file, type, interface, port, function, method, field, and variable has one bounded responsibility, stateable in one sentence; if it cannot be, fix the abstraction. Never use `util.go`, `helpers.go`, `handler.go`, or `types.go`. Reject umbrella names such as Store, Manager, Service, Handler, Util, or catch-all domain names like Distribution*.

Naming — symmetry: When one variant is qualified, qualify every sibling the same way. Parallel implementations get parallel names (`elasticsearchSearchOnce`, `manticoreSearchOnce`); never leave one sibling bare.

Naming — boundaries: Name a package `<subject><artifact>`: the head is its principal exported type and is never an activity; the subject is the domain object it concerns. Repeat the parent package's name when the name is read without its path. The name is an exhaustive predicate — everything inside is implied by it, nothing inside falls outside it. Two parallel vocabularies in one unit (two outcome enums, two receipt types) is a boundary no name can cover. If the best head noun is a nominalized verb, fix the boundary, not the name.

Naming — style: Name a thing for what it is in the problem domain, not how it is built or what it is for. Strip implementation terms (count, map, hash, digest, buffer) and destination terms (shared, peer, abstract, response); keep the domain noun. Spell names in full; abbreviation is not allowed. Confine protocol- and transport-specific vocabulary to the edge that translates to and from it; inner code speaks plain domain language.

Naming — derivation: A function that returns a value is named for the value, never for the making of it: a domain noun phrase joined to its source by a preposition, reading at the call site as the value with its arguments as the subject (`replicas.HoldersOf(tx, posting)`, `interval.widenedFrom(previousInterval)`). The preposition names the relation: `Of` an attribute the subject has, `From` a value derived from it, `For` a value serving it; a misstated relation fails the name. The preposition binds to the first parameter; the parameters after it are the context the value is computed against, and context and transaction parameters are never the subject. A value with no single subject takes no preposition. A function that acts keeps its verb. Reserve `new<Type>` for a collaborator assembled from injected dependencies, never for a computed value.

Naming — subject: A bare adjective or participle is never a name. Every name carries the noun it qualifies: `duePostings`, not `due`; `recordedHolders`, not `recorded`; `postingsAcceptedByPeer`, not `accepted`. The comma-ok idiom keeps `found`.

Naming — predicates: A name that yields a boolean asks a question about its subject and reads as one where it is used: `isEligible(peer)`, `peer.IsReachable(...)`, `acceptsRemoteIndex(seed)`, `responsible.contains(peer)`. `is`, `has` and `can` are for a state of being. A boolean field takes its subject from its receiver: `answer.Accepted`.

Vocabulary: A package speaks one vocabulary, taken from the running system's own documents, never a metaphor laid over it. Transport words stay at the transport edge. One fact carries one word along its whole path — wire, ledger, config, doc — and one word carries one meaning per file. A boolean parameter that selects the rule is two functions with symmetric names.

Dependencies: Record each new third-party dependency in its own ADR before use.

Version pinning: Pin all versions. Runtime deps: go.mod. Build/lint tools: Go tool directives in go.mod. make verify uses only pinned tools, never PATH versions.

Testing: Code lands with tests. make verify runs tests and coverage and is the only gate; a change is done only when it is green and total coverage stays at or above the threshold.

Coverage: If coverage drops, first remove or refactor code: delete dead or defensive-only code, collapse unexercised branches, or replace several paths with one covered path. Add tests only for required behavior; filler tests written only to raise coverage fail the change.
