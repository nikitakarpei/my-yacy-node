Structure: new capability → new unit (file or package) on the smallest seam; behavior change → edit in place. A unit owns durable state, a protocol endpoint, or a value type and the rule computing it; a stage of a procedure is not ownable, and stages that only run in sequence are one unit. Split a unit when responsibilities diverge; collapse a seam when it couples units tightly.

Shared state: the service that owns a stream, bucket, or table is the only one that creates it. A contract package holds vocabulary — names, subjects, keys — never provisioning. A consumer opens what it reads and adds no precondition check.

Abstraction: a function orchestrates or it works, never both. An orchestrator only names steps and reads as the scenario in plain language; log attributes, struct assembly, and error wrapping are detail. A call chain deeper than one step means the middle level speaks its own vocabulary and needs its own unit.

File layout: one level of abstraction per file. The entry point comes first, then each function it names in call order, each followed the same way; a helper called twice goes under its first caller. A type's methods and the private functions only it calls live in the file that declares it, never split.

Naming: read doc/naming.md before naming a package, type, or derived value. Always — name the domain thing, not its construction or destination; spell in full; every name carries its noun (`duePostings`, not `due`); never `util.go`, `helpers.go`, `handler.go`, `types.go`, or umbrella names such as Store, Manager, Service, Handler, Util.

Naming — a name stands alone: a name carries its full meaning without help from what surrounds it. Never leave a word out because the package, file, type, function, or a nearby argument already suggests it. The reader meets the name by itself — at a call site, in a log line, in an error — where none of that context is in view.

Naming — a collision renames both: when the right name for new code is already taken, never settle for a worse one. Rename both — give the new code the name it needs, and give the existing code the name that fits what it really holds. Code that is already there never lowers the quality of the code being added.

Naming — derivation: a function that returns a value is named for the value, never for the making of it — a domain noun phrase joined to its source by a preposition that names the relation: `Of` an attribute the subject has, `From` a value derived from it, `For` a value serving it (`replicas.HoldersOf(tx, posting)`, `interval.widenedFrom(previousInterval)`). It reads at the call site as the value, with its arguments as the subject. A function that acts keeps its verb.

Naming — a hard name means a weak unit: if a good name will not come, or the user says a name is unreadable, treat it as a design problem, not a wording problem. Do not look for a better word. Look at what the code does and what it owns. If the name has to hide a side effect, or if it names the returned value while the body also reserves, starts, or retries something, then the code is doing a job that no unit owns yet. Give that job its own unit, then name it.

Vocabulary: one fact carries one word along its whole path — wire, ledger, config, doc — and one word carries one meaning per file. Transport words stay at the transport edge. A boolean parameter that selects the rule is two functions with symmetric names.

Single source of truth: every fact lives in exactly one place — a constant, config, or doc. Never restate it in comments, errors, logs, or a second doc.

Comments: only the forms allowed here, plus any the user approves for a specific change. Use naming and structure instead. Allowed: godoc package docs, and `// TECHDEBT:` markers.

Tech debt: when you read code that breaks a rule in this file, mark it where you found it with a single `// TECHDEBT: <which rule, and what breaks it>` line, even when the file is outside the task. A marker names the rule and the breach, never a remedy. Mark it once, on the declaration that owns the violation. Never mark code you are already changing — fix that instead.

Docs: each doc is self-contained, at most 80 lines unless the user approves an exception, with paragraphs of at most five lines, written in ASD-STE100 Simplified Technical English. It covers only what someone using the system needs — no implementation detail, no rationale — and every sentence must be true against the running system.

Failure: a unit stays on the happy path and returns no error to its consumer. The implementation that knows what went wrong reports it to its own observer, in its own vocabulary, and reports only what it can see; a decorator never assumes what the implementation it wraps reports, and duplicated facts are the honest cost. The consumer acts on a reported failure only if it has something to decide. Returning an error is failing fast, and needs the user's approval for that case. Wiring fails fast: a unit that opens a stream, bucket, or table it does not own returns the error unchanged, and the service does not start.

Logging: nothing fails silently. Happy paths DEBUG, recoverable failures WARN, failures that need an operator to intervene ERROR.

Arch-lint: the composition root may use anything; every other component lists what it may use. A change that allows more — a new common component or vendor, a new edge, a wider glob — names the dropped rule in the commit.

Dependencies: record each new third-party dependency in its own ADR before use. Pin all versions — runtime deps in go.mod, build and lint tools in Go tool directives; make verify uses only pinned tools, never PATH versions.

Testing: code lands with tests. A test asserts the observable behavior of one cohesive unit, never its internals, which may change while the behavior stays the same. Every test file declares `package <name>_test` and reaches the code through its exported surface only. A test that needs an unexported identifier has found a unit that is not extracted yet; extract it, and never widen an exported surface to let a test in. A test that spans packages lives under `test/`, in a directory that holds no code. make verify is the only gate; a change is done only when it is green. If coverage drops, first delete dead or defensive-only code or collapse unexercised branches; filler tests written only to raise coverage fail the change.
