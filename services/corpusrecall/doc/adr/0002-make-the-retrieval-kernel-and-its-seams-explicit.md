# 2. Make the retrieval kernel and its seams explicit

Status: accepted

## Context

The retrieval engine and the adapters around it sit at the same level. The
layout does not say which package holds retrieval logic.

The engine depends on the mechanisms it should be independent of. It receives a
storage revision number as the record of a disposed page. Interfaces at the
broker edge carry broker types, and exist so that tests can replace them.

No unit owns a representation. A representation kind, its content, its source,
and its wire encoding live in four packages. The composition root joins them and
trusts the join without a check. A second kind repeats the spread.

The specification requires more than the layout gives. The search index and the
broker are each replaceable with no change to retrieval. The retrieval port
admits more than one edge adapter over one core. Today there is one edge, and it
belongs to the composition root.

`yacycrawler` answered the same question in its ADR 11.

## Decision

The kernel is the retrieval engine. It holds one namespace of its own. It speaks
the domain language, and it declares every interface it needs beside the code
that calls it.

Each extension point the kernel opens is a seam. A seam is a namespace that
holds plugins and nothing else. A plugin implements the interface of one kernel
package. The kernel depends on no plugin.

A plugin owns its mechanism whole. It holds its own broker, storage, or protocol
handle. No interface stands between a plugin and the library it adapts.

A representation plugin owns its kind, its content, and its source. It imports
no API package. A recall receiver owns the contract form of every kind it
serves. At assembly it checks that each served kind has a form, and it refuses
to start when one is missing.

The seams are page representations, redirect resolvers, disposed pages, crawl
order placers, progress observers, and recall receivers. Four of these names
carry the meaning they already carry in `yacycrawler`.

The architecture linter enforces the boundary. Convention does not.

## Consequences

- A new representation kind is one plugin and one contract form in each recall
  receiver. It changes no kernel file.
- A second edge adapter is one plugin. It changes no kernel file.
- The kernel no longer changes when a mechanism changes.
- The service holds more directories than a flat layout needs.
- `corpusrecall` becomes the third service on this layout. Five stay flat.
