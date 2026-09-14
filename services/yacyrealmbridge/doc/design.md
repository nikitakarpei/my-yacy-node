# yacyrealmbridge — Solution design (draft)

Draft record for step 3 of `doc/design-process.md`: boxes, boundary contracts, decisions with reasons. Not stressed yet.

## Mechanism

The bridge is network address translation for YaCy peers. It holds one address of its own in each realm and a
pool of ports on it. A confirmed native address in realm A gets one peer address table entry: that native address
bound to a translated address, a port on the bridge address in B. A seed that crosses into B carries the translated
address in place of the native one; nothing else changes. The bridge greets confirmed peers of B with each translated
seed under the peer's own hash; the greeted peer calls back, is forwarded to the native address, and records the seed as senior.

A request that arrives on a translated address goes to the native address of its table entry. Every seed the
request or its answer carries is translated for the realm it enters; in a hello answer `yourip` becomes the
address the bridge saw the caller on. The bridge has no peer identity of its own, and it never reads a peer hash
as identity: an entry binds two addresses, and the hash rides inside the seed unchanged. A seed a request carries
is a discovery in the caller's realm.

## Units

| Package | Owns | Contract |
|---|---|---|
| `addressrealm` | The two realm names and a crossing between them. | `Name`, `Pair.OtherOf(Name) Name`, `Crossing{From, Into}` |
| `realmpeers` | The native peers of one realm, keyed by native address: the seed each advertised, and confirmed reachability. Known peers are durable; reachability lives in memory. One instance per realm. | `Discover(seeds)`, `ConfirmReachable(address, seed)`, `ConfirmUnreachable(address)`, `IsConfirmedNative(address) bool`, `ConfirmedSeedAt(address) Optional[Seed]`, `KnownAddresses() []NetworkAddress`, `MostRecentlyConfirmedSeeds(limit) []Seed` |
| `realmpeerdiscovery` | The cycle of one realm: reads its seed lists, probes each known native address, reports each outcome to `realmpeers`, and binds each confirmed native address in the peer address table. | `RunOnce(ctx)`, `Run(ctx)` |
| `peeraddresstable` | The durable table: a native address in one realm bound to a translated address in the other, drawn from that realm's address pool. An entry lives until its retention ends. | `Bind(nativeRealm, nativeAddress) (Entry, ok)`, `EntryAtTranslated(address) Optional[Entry]`, `EntryOfNative(nativeRealm, address) Optional[Entry]`, `Entry{NativeRealm, NativeAddress, TranslatedAddress}` |
| `seedtranslation` | The seed one seed becomes when it enters a realm, and the rule: an address confirmed native there stays; a native address bound in the table becomes its translated address; a seed left with no address drops. | `TranslatedSeedFor(seed, into) Optional[Seed]` |
| `yacypeerproxy` | The proxied `/yacy/` peer-protocol endpoints, one file per path. Each endpoint resolves the translated address the request arrived on to its entry, applies the rules of its path for that crossing, answers a refusal in place in its path's own vocabulary, and otherwise forwards to the native address with the fields that carry seeds and `yourip` translated. | `NewMux(...) *http.ServeMux`, served on every address of the pool |
| `seedannouncement` | The cycle that greets confirmed peers of one realm with the translated seed of each bound peer of the other realm, under that peer's own hash, and reports each contact outcome. | `Run(ctx)` |
| `yacyseedlist`, `peerlivenesswire`, `peergreetwire` | Wire units: read one seed list; probe one address with `query.html?object=rwicount` and the network name; send one hello and read its answer. | As in `yacydhtsearch` and `yacynode` |
| `<unit>observers/prometheus`, `<unit>observers/applog` | Metrics and logs of one unit, one observer interface per unit. | Per unit |
| `cmd/yacyrealmbridge` | Composition root: configuration, the vault, one outbound client per realm, one listener per pool address, the cycles. | `RunService(ctx, cfg, registry)` |

## Decisions

**An address is identity; a hash is payload.** Anyone can claim a hash, and an address is what the bridge
confirmed it can reach. A peer that moves gets a new entry, and its old entry ends with the retention. The
specification line "stable across peer moves" then needs to read "stable while the native address holds".

**A translated address is the bridge address in the receiving realm plus one port from a configured pool, and
the bridge listens on the whole pool from start.** A YaCy peer reaches another by `IP` and `Port` only. A hello
receiver compares the seed's address with the connecting address, and the bridge greets from that same address.
A pool address with no entry answers as an unavailable peer, so no listener opens at run time.

**A translated seed replaces `IP` and `Port`, drops `IPs` and `PortSSL`, and clears the SSL-available flag.**
A translated address speaks plain HTTP only. This reads the SSL flag as reachability, not capability; the
specification lists capabilities as preserved, so the owner must confirm this reading.

**Confirmation is a liveness probe; announcement is a hello.** Withdrawal needs a probe that does not depend on
greeting anyone, and the bridge has no seed of its own to hello with. Discovery confirms, then binds.

**One translation rule covers the return path, the dual-homed peer, and dropping.** A seed carrying an address
confirmed native in the receiving realm keeps that address: this returns a translated address to its native one
and leaves a peer native to both realms alone. A bound native address becomes its translated one. Any other seed drops.

**A carried seed that does not translate: required field refuses the request, optional field or answer list
drops the seed.** An entry whose native address is unconfirmed answers the same way. Refusal here is an
unavailable peer, HTTP 503: a failed contact is the case every YaCy peer already handles by trying again later.

**Each proxied endpoint owns its rules and its refusal answers; there is no rule engine.** The endpoint of a path
reads the configuration of its own crossing, for example refuse `transferRWI` into one realm with `result=not_granted`
and a pause, and answers in its path's vocabulary. The mux mounts only the paths `yacyproto` names. A foreign
network gets the hello answer YaCy gives, `yourtype=virgin`, and HTTP 403 elsewhere.

**Durable state is one vault with two collections: known peers per realm, and the peer address table.** At start
each realm runs `RunOnce` to reconfirm its known addresses, then the listeners and the announcement cycle start. An
entry ends when its native address stays unreachable past the configured retention, or a finite pool fills with departed peers.

**Only fields that carry seeds and `yourip` are re-encoded.** Seeds sit in hello `seed`, search `myseed`, and hello answer `seed0..N`. Every other request or answer, `urls.xml` included, crosses byte for byte.

**Wire units are lifted, not copied.** `yacyseedlist` and `peerlivenesswire` move from `yacydhtsearch` to
`libraries/`; the hello exchange of `yacynode`'s `peerannouncement` becomes `peergreetwire` there too.

## Residual risks and open points

* A rogue peer that claims another peer's hash reaches the other realm under it; YaCy keys peers by hash.
* The seed codec carries only the columns `yacymodel` names; a column it does not name does not cross.
* The spread of a translated seed in the receiving realm relies on YaCy gossip beyond the greeted peers.
* One realm's outbound client must leave through that realm: a bind address or an egress proxy per realm.
* ADRs to write before code: record decisions, the embedded storage engine, and the lifted wire libraries.
