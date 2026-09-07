# Naming

Every package, file, type, function, method, field, and variable has one
responsibility, stateable in one sentence. Its name states that responsibility
exactly: nothing inside falls outside the name, and nothing the name promises is
missing. Two vocabularies under one name are two units.

## Packages

A package name is `<subject><head>`: `postingcourier`, `postingoffer`,
`postingreplicas`. Siblings repeat the subject. A package that owns nothing takes
no head, and is not a package.

Port implementations live under a plural directory named for the port, which
holds no code. Each one is named for its technology or variant and takes `New`:
`pagefetchers/http`, `vaultengines/bolt`. The interface belongs to the consumer.

A package that speaks a wire protocol carries the protocol in its name, and never
reads as pure domain vocabulary.

## Words

Name the domain thing, not how it is built or where it goes. Strip implementation
terms (count, map, hash, digest, buffer) and destination terms (shared, peer,
abstract, response). Spell every word in full. Transport vocabulary stays in the
package that translates it.

## Phrases

A name reads as a complete phrase and stands alone. Skip no word that the
package, the type, or the code beside it supplies: `amountOfItemsAcrossAnswers`,
not `answeredItems`.

Every name carries its noun: `duePostings`, not `due`. A bare adjective or
participle is never a name; the comma-ok idiom keeps `found`.

A name states the state the value is in, not one it reaches later: `chosenPeers`
before the call that asks them.

A quantity takes `amountOf` for things, and `sumOf` for values that add.

## Derivation

A function that returns a value is named for the value. A function that acts
keeps its verb.

A preposition binds the value to the first parameter: `Of` an attribute the
subject has, `From` a value derived from it, `For` a value that serves it.
Context and transaction parameters are never the subject. A value with no single
subject takes no preposition.

A derivation names the domain fact, never the conversion, and never only its
source.

`new<Type>` is reserved for a collaborator assembled from injected dependencies.

## Symmetry

Parallel things get parallel names: `elasticsearchSearchOnce`,
`manticoreSearchOnce`. When one variant is qualified, every sibling is qualified
the same way.
