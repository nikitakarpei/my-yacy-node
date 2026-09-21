# Naming

Every package, file, type, function, method, field, and variable has one
responsibility, stateable in one sentence. Its name states that responsibility
exactly. Two vocabularies under one name are two units.

## Packages

A package name is `<subject><head>`: `postingcourier`, `postingoffer`,
`postingreplicas`. Siblings repeat the subject. A package that owns nothing is
not a package.

Port implementations live under a plural directory named for the port, which
holds no code. Each is named for its technology or variant and takes `New`:
`pagefetchers/http`, `vaultengines/bolt`. The interface belongs to the consumer.

A package that speaks a wire protocol carries the protocol in its name.

## Words

Name the domain thing, not how it is built or where it goes: `duePostings`,
not `postingBuffer` or `postingsToSend`.

One word has one meaning in a file: `order` is a purchase or a sequence,
never both.

## Phrases

Every call reads as one phrase at its call site, with the argument
expressions as written and without its body: `price := priceOf(item)`. The
package qualifier and non-domain arguments (`ctx`, `tx`, `t`) are not read.

One domain argument completes a trailing preposition. A second one that owns
the rule becomes the receiver: `catalog.priceOf(item)`, not
`priceOfItemIn(item, catalog)`.

Every name carries its noun: `duePostings`, not `due`. Only a boolean that
reports the outcome of its own call has no noun: `found`, `wasReplaced`.

A name states the state the value is in now: `chosenPeers` before the call
that asks them.

A quantity takes `amountOf` for things and `sumOf` for values that add.

## Length

Keep a word only if the reader would assume something else without it.

Drop: an article that picks nothing out (`sumOfPrices`); a type or package in
view at every use; an actor the code cannot tell apart; a state the value
always has there (in a loop over accepted orders, each one is `order`).

Keep: the noun; a qualifier that separates two things in one file
(`priceOfItem(item)` beside `priceOfItems(items)`); an article that changes
the meaning (`itemAUserChose`: any user); a preposition that names the relation.

A name past four words is not shortened by wording. It names two facts or two
jobs: split it into two units or two values.

## Derivation

Name a function for the value it returns. A function that returns nothing keeps
its verb.

A name that ends with a preposition is completed by its domain argument:
`HoldersOf(posting)`, `widenedFrom(previousInterval)`,
`spreadWithin(queryBudget)`. A name whose value comes from all its arguments
has no preposition.

`new<Type>` is reserved for a collaborator assembled from injected dependencies.

## Symmetry

Parallel things get parallel names: `elasticsearchSearchOnce`,
`manticoreSearchOnce`. When one variant is qualified, every sibling is.
