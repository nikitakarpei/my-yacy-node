Structure: new capability → new unit (file or package) on the smallest seam; behavior change → edit in place. A unit owns durable state, a protocol endpoint, or a value type and the rule computing it; a stage of a procedure is not ownable, and stages that only run in sequence are one unit. Split a unit when responsibilities diverge; collapse a seam when it couples units tightly.

Shared state: the service that owns a stream, bucket, or table is the only one that creates it. A contract package holds vocabulary — names, subjects, keys — never provisioning. A consumer opens what it reads and passes the broker's error through; it adds no precondition check.

Abstraction: a function orchestrates or it works, never both. An orchestrator only names steps and reads as the scenario in plain language; log attributes, struct assembly, and error wrapping are detail. A call chain deeper than one step means the middle level speaks its own vocabulary and needs its own unit.

File layout: one level of abstraction per file. The entry point comes first, then each function it names in call order, each followed the same way; a helper called twice goes under its first caller. A type's methods and the private functions only it calls live in the file that declares it, never split.

Naming: read doc/naming.md before naming a package, type, or derived value. Always — name the domain thing, not its construction or destination; spell in full; every name carries its noun (`duePostings`, not `due`); never `util.go`, `helpers.go`, `handler.go`, `types.go`, or umbrella names such as Store, Manager, Service, Handler, Util.

Naming — derivation: a function that returns a value is named for the value, never for the making of it — a domain noun phrase joined to its source by a preposition that names the relation: `Of` an attribute the subject has, `From` a value derived from it, `For` a value serving it (`replicas.HoldersOf(tx, posting)`, `interval.widenedFrom(previousInterval)`). It reads at the call site as the value, with its arguments as the subject. A function that acts keeps its verb.

Vocabulary: one fact carries one word along its whole path — wire, ledger, config, doc — and one word carries one meaning per file. Transport words stay at the transport edge. A boolean parameter that selects the rule is two functions with symmetric names.

Single source of truth: every fact lives in exactly one place — a constant, config, or doc. Never restate it in comments, errors, logs, or a second doc.

Comments: no comments without explicit user approval. Use naming and structure instead. Godoc package docs are allowed.

Docs: each doc is self-contained, at most 80 lines unless the user approves an exception, with paragraphs of at most five lines, written in ASD-STE100 Simplified Technical English. It covers only what someone using the system needs — no implementation detail, no rationale — and every sentence must be true against the running system.

Logging: nothing fails silently. Happy paths DEBUG, recoverable failures WARN, failures that need an operator to intervene ERROR.

Arch-lint: the composition root may use anything; every other component lists what it may use. A change that allows more — a new common component or vendor, a new edge, a wider glob — names the dropped rule in the commit.

Dependencies: record each new third-party dependency in its own ADR before use. Pin all versions — runtime deps in go.mod, build and lint tools in Go tool directives; make verify uses only pinned tools, never PATH versions.

Testing: code lands with tests. A test asserts the observable behavior of one cohesive unit, never its internals, which may change while the behavior stays the same. Every test file declares `package <name>_test` and reaches the code through its exported surface only. A test that needs an unexported identifier has found a unit that is not extracted yet; extract it, and never widen an exported surface to let a test in. A test that spans packages lives under `test/`, in a directory that holds no code. make verify is the only gate; a change is done only when it is green. If coverage drops, first delete dead or defensive-only code or collapse unexercised branches; filler tests written only to raise coverage fail the change.
