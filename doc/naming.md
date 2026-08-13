# Naming

Every package, file, type, interface, port, function, method, field, and variable has one bounded responsibility, stateable in one sentence. The name is an exhaustive predicate over it: everything inside is implied by the name, and nothing inside falls outside it. Two parallel vocabularies in one unit, such as two outcome enums or two receipt types, are a boundary that no name covers; split the unit.

## Packages

A package name has the form `<subject><head>`. The subject is the domain object the package concerns, repeated by every sibling that shares it, because the name is read without its path: `postingcourier`, `postingoffer`, `postingreplicas`. The head names what the package owns — `searchresult`, `pageintake`, `ordersettlement`. A package that owns nothing has no head to take, and is not a package.

Implementations of a port live under a plural directory that carries the port name and holds no code. Each is a package named for its technology or variant, and takes `New` (`pagefetchers/http`, `vaultengines/bolt`, `recrawlrules/dueaftergrace`). The interface belongs to the consumer that calls it.

## Style

Name a thing for what it is in the problem domain, not for how it is built or what it is for. Strip implementation terms (count, map, hash, digest, buffer) and destination terms (shared, peer, abstract, response); keep the domain noun. Spell names in full; abbreviation is not permitted. Protocol and transport vocabulary stays at the edge that translates to and from it.

## Derivation

The preposition binds to the first parameter. The parameters after it are the context the value is computed against; context and transaction parameters are never the subject. A value with no single subject takes no preposition. A misstated relation fails the name.

`new<Type>` is reserved for a collaborator assembled from injected dependencies, never for a computed value.

## Subjects and predicates

A bare adjective or participle is never a name. Every name carries the noun it qualifies: `duePostings`, not `due`; `recordedHolders`, not `recorded`; `postingsAcceptedByPeer`, not `accepted`. The comma-ok idiom keeps `found`.

A name that yields a boolean asks a question about its subject and reads as one where it is used: `isEligible(peer)`, `peer.IsReachable(...)`, `acceptsRemoteIndex(seed)`, `responsible.contains(peer)`. `is`, `has`, and `can` are for a state of being. A boolean field takes its subject from its receiver: `answer.Accepted`.

## Symmetry

When one variant is qualified, every sibling is qualified the same way. Parallel implementations get parallel names: `elasticsearchSearchOnce`, `manticoreSearchOnce`. No sibling is left bare.
