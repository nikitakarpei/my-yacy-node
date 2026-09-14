# yacyrealmbridge — Solution design (draft)

Draft record for step 3 of `doc/design-process.md`: boxes, boundary contracts, decisions with reasons. Not stressed yet.

## Mechanism

The bridge holds one address of its own in each realm. A native peer of realm A gets one projection into
realm B: a port on the bridge address in B, and the peer's own seed with only its reachability fields
replaced by that endpoint. The bridge greets confirmed native peers of B with the projected seed under the
peer's own hash. A greeted peer calls the projected endpoint back, the bridge forwards the call to the
native peer, and the greeted peer records the projection as a senior peer and gossips it. B into A is symmetric.

A request that reaches a projection endpoint goes to the confirmed native address of that projection's
peer. Every seed the request or its answer carries is translated for the realm it enters. In a hello answer
the `yourip` field becomes the address the bridge saw the caller on. The bridge has no peer identity of its
own: it speaks only as the peers it projects. A seed a request carries is a discovery in the caller's realm.

## Units

| Package | Owns | Contract |
|---|---|---|
| `addressrealm` | The two realm names and a crossing between them. | `Name`, `Pair.OtherOf(Name) Name`, `Crossing{From, Into}` |
| `realmpeers` | The native peers of one realm: known seeds, native addresses, confirmed reachability. Known peers are durable; reachability lives in memory. One instance per realm. | `Discover(seeds)`, `ConfirmReachable(hash, address)`, `ConfirmUnreachable(hash)`, `ConfirmedNativeSeedOf(hash) Optional[Seed]`, `MostRecentlyConfirmedPeers(limit) []Seed`, `KnownPeers() []Seed` |
| `realmpeerrefresh` | The cycle of one realm: reads its seed lists, probes each known peer at its native address, reports each outcome to `realmpeers`, and projects each confirmed peer into the other realm. | `RunOnce(ctx)`, `Run(ctx)` |
| `peerprojections` | The durable ledger: for each receiving realm, peer hash to endpoint, drawn from that realm's endpoint pool. The ledger keeps an endpoint until its retention ends. | `Project(hash, into) (Projection, ok)`, `ProjectionAt(endpoint) Optional[Projection]`, `ProjectionOf(hash, into) Optional[Projection]`, `Projection{PeerHash, NativeRealm, Endpoint}` |
| `seedtranslation` | The value one seed becomes when it enters a realm, and the rule: native there gives the native seed; projected there gives the projected seed; neither drops the seed. | `TranslatedSeedFor(seed, into) Optional[Seed]` |
| `boundaryrules` | The operator rules: one crossing, a set of peer-protocol paths, a verdict, and the refusal a refusing rule names. | `VerdictFor(crossing, path) Verdict`, `Verdict{Admit}` or `Verdict{Refuse, Refusal}` |
| `refusalanswers` | The answer the bridge gives in place of the native peer, in the vocabulary of the refused path. | `AnswerFor(path, refusal) Answer{Status, Message}` |
| `crossingwire` | Carries one admitted peer request across a crossing to the native address and its answer back. It decodes and re-encodes only the fields that carry seeds and `yourip`; every other byte passes through. | `Cross(ctx, crossing, nativeAddress, request) Answer` |
| `projectionendpoints` | The listeners on every projection endpoint of one realm. Each request resolves its endpoint to a projection, passes the built-in gates, takes the rule verdict, and is refused in place or crossed. | `http.Handler` per endpoint; `Serve(ctx)` |
| `projectionannouncement` | The cycle that greets confirmed native peers of the receiving realm with each projection's translated seed under the peer's own hash, and reports each contact outcome. | `Run(ctx)` |
| `yacyseedlist`, `peerlivenesswire`, `peergreetwire` | Wire units: read one seed list; probe one address with `query.html?object=rwicount` and the network name; send one hello and read its answer. | As in `yacydhtsearch` and `yacynode` |
| `<unit>observers/prometheus`, `<unit>observers/applog` | Metrics and logs of one unit, one observer interface per unit. | Per unit |
| `cmd/yacyrealmbridge` | Composition root: configuration, the vault, one outbound client per realm, the listeners, the cycles. | `RunService(ctx, cfg, registry)` |

## Decisions

**Endpoint = bridge address in the receiving realm + one port from a configured range.** A YaCy peer reaches
another only by `IP` and `Port`; there is no host-based multiplexing. A hello receiver compares the seed's
address with the connecting address, and the bridge greets from that same address, so the two agree.

**A projected seed replaces `IP`, `IPs`, `Port`, drops `PortSSL`, and clears the SSL-available flag.** A
projection endpoint speaks plain HTTP only. This reads the SSL flag as reachability, not capability; the
specification lists capabilities as preserved, so the owner must confirm this reading.

**Confirmation is a liveness probe; announcement is a hello.** Withdrawal needs a probe that does not depend on
greeting anyone, and the bridge has no seed of its own to hello with. `realmpeerrefresh` confirms, then projects.

**One translation rule covers reverse translation, suppression, and dropping.** A seed whose hash is native
to the receiving realm becomes that native seed: this returns a projection to its origin and suppresses the
projection of a peer native to both realms. A seed with a projection becomes the projected seed. Any other seed drops.

**A carried seed that does not translate: required field refuses the request, optional field or answer list
drops the seed.** A withdrawn projection answers the same way. Refusal here is an unavailable peer, HTTP 503:
a failed contact is the case every YaCy peer already handles by trying again later.

**Built-in gates run before operator rules, and no matching rule admits.** The gates: the path is one of the
peer-protocol paths `yacyproto` names, the request names the configured network, and the seed parses. A
foreign network gets the hello answer YaCy gives, `yourtype=virgin`, and HTTP 403 on other paths.

**Durable state is one vault with two collections: known peers per realm, and the projection ledger.**
Endpoints stay stable across restarts and peer moves because the ledger keys on the peer hash. At start each
realm runs `RunOnce` to confirm its known peers, then the listeners and the announcement cycle start.

**An endpoint is released when its peer has been unreachable longer than the configured retention.** Without
release a finite pool fills with departed peers and the bridge refuses every new peer, for ever.

**Only fields that carry seeds and `yourip` are re-encoded.** Seeds sit in hello `seed`, search `myseed`, and
hello answer `seed0..N`. Every other request or answer, `urls.xml` included, crosses byte for byte, so
fields `yacyproto` does not model still cross.

**Wire units are lifted, not copied.** `yacyseedlist` and `peerlivenesswire` move from `yacydhtsearch` to
`libraries/`; the hello exchange of `yacynode`'s `peerannouncement` becomes `peergreetwire` there too.

## Residual risks and open points

* The seed codec carries only the columns `yacymodel` names; a column it does not name does not cross.
* The spread of a projection in the receiving realm relies on YaCy gossip beyond the greeted peers.
* One realm's outbound client must leave through that realm: a bind address or an egress proxy per realm.
* ADRs to write before code: record decisions, the embedded storage engine, and the lifted wire libraries.
