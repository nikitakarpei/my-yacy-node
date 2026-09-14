# yacyrealmbridge — Solution design (draft)

Draft record for step 3 of `doc/design-process.md`: boxes, boundary contracts, decisions with reasons. Stressed once by the owner.

## Mechanism

The bridge is network address translation for YaCy peers. It holds one address of its own in each realm and a
pool of ports on it. A confirmed native address in realm A gets one peer address table entry: that native address
bound to a translated address, a port on the bridge address in B. A seed that crosses into B carries the translated
address as `IP`, every other address B gossip already carries for that peer as `IPs`, and a tag that marks it as
translated; nothing else changes. Peers keep the addresses that answer and drop the rest, so B heals itself when a bridge dies.

The bridge greets confirmed peers of B with each translated seed under the peer's own hash. The greeted peer calls
the translated address back, the bridge forwards the call to the native address, and the greeted peer records the
seed as senior. A request that arrives on a translated address goes to the native address of its entry, and every
seed it or its answer carries is translated for the realm it enters; in a hello answer `yourip` becomes the address
the bridge saw the caller on. The bridge has no peer identity of its own and speaks only as the peers it forwards.

## Units

| Package | Owns | Contract |
|---|---|---|
| `addressrealm` | The two realm names and a crossing between them. | `Name`, `Pair.OtherOf(Name) Name`, `Crossing{From, Into}` |
| `realmpeers` | The native peers of one realm, keyed by native address: candidate addresses, the seed each peer stated about itself, and confirmed reachability. Known addresses are durable; reachability lives in memory. One instance per realm. | `Discover(address)`, `ConfirmStated(address, seed)`, `ConfirmUnreachable(address)`, `ConfirmedSeedAt(address) Optional[Seed]`, `IsConfirmedNativeHash(hash) bool`, `KnownAddresses() []NetworkAddress`, `MostRecentlyConfirmedSeeds(limit) []Seed` |
| `realmpeerdiscovery` | The cycle of one realm: reads its seed lists, greets each known address with one translated seed of the other realm, stores the seed the answer states, reports each outcome, and binds each confirmed native address. A tagged seed, or one whose hash is native to the other realm, goes to `gossipedpeeraddresses` instead. | `RunOnce(ctx)`, `Run(ctx)` |
| `gossipedpeeraddresses` | The bridge's view of the addresses one realm's gossip currently carries for a peer it forwards, by hash. Several bridges may connect the same realms without knowing each other: each owns only its own doors, and learns the others from gossip. In memory only; an address not heard for the configured cycles is forgotten. | `Learn(realm, seed)`, `AddressesOf(realm, hash) []NetworkAddress` |
| `peeraddresstable` | The durable table: a native address in one realm bound to a translated address in the other, drawn from that realm's address pool. An entry lives until its retention ends or the bridge ends it. | `Bind(nativeRealm, nativeAddress) (Entry, ok)`, `EntryAtTranslated(address) Optional[Entry]`, `EntryOfNative(nativeRealm, address) Optional[Entry]`, `End(entry)`, `Entry{NativeRealm, NativeAddress, TranslatedAddress}` |
| `seedtranslation` | The seed one seed becomes when it enters a realm, and the rule: a hash confirmed native there drops; a bound native address becomes `IP` = its translated address, `IPs` = the gossiped addresses, plus the tag; any other seed drops. | `TranslatedSeedFor(seed, into) Optional[Seed]` |
| `peerrequestforwarding` | The `/yacy/` peer-protocol endpoints served on every pool address, one file per path. Each endpoint resolves the arriving translated address to its entry, applies the rules of its path for that crossing, refuses in place in its path's own vocabulary, or forwards to the native address with the seed fields and `yourip` translated. | `NewMux(...) *http.ServeMux` |
| `seedannouncement` | The cycle that greets confirmed peers of one realm with the translated seed of each bound peer of the other realm, under that peer's own hash, and reports each contact outcome. | `Run(ctx)` |
| `yacyseedlist`, `peergreetwire`, `peerwire` | Wire units, lifted to `libraries/` rather than copied: read one seed list (from `yacydhtsearch`); send one hello and read its answer (the hello exchange of `yacynode`'s `peerannouncement`); post one form to one peer address and read the message back (`yacynode`'s `peerwire`). | `New(client, ...)` each |
| `<unit>observers/prometheus`, `<unit>observers/applog` | Metrics and logs of one unit, one observer interface per unit. | Per unit |
| `cmd/yacyrealmbridge` | Composition root: configuration, the vault, one outbound client per realm, one listener per pool address, the cycles. | `RunService(ctx, cfg, registry)` |

## Decisions

**An address is identity; a hash is payload.** Anyone can claim a hash; an address is what the bridge confirmed it
can reach. A peer that moves gets a new entry, and its old one ends with the retention, so the specification line
"stable across peer moves" reads "stable while the native address holds". The hash serves one purpose: to recognise,
in one realm's gossip, a peer the bridge already forwards from the other realm.

**A translated address is the bridge address in the receiving realm plus one port from a configured pool, and the
bridge listens on the whole pool from start.** A YaCy peer reaches another by `IP` and `Port` only, and a hello receiver
compares the seed's address with the connecting one, so the bridge greets from that same address. A pool address with no entry answers as an unavailable peer.

**A translated seed sets `IP`, `Port`, `IPs`, and the tag; drops `PortSSL`; clears the SSL-available flag.** A
translated address speaks plain HTTP only. `IPs` carries the doors other bridges opened, as the receiving realm's
gossip states them, so peers fail over between bridges on their own. The tag is one fixed label in `Tags`, which
YaCy only displays. The specification must gain the `IPs` and tag rules, and confirm the SSL flag reading.

**A door is never a peer.** A seed heard in a realm is no native candidate there when it carries the tag, or when
its hash is confirmed native in the other realm. Otherwise one bridge builds a door onto another bridge's door, and
the chain never breaks because it answers. An entry whose hash later proves native to the other realm ends.

**A native address states its own seed; the bridge repeats no rumour.** A seed from an operator seed list is
trusted as stated. A seed from gossip or a carried request only names a candidate address. Discovery greets that
address with one translated seed of the other realm, and the `seed0` of the answer is the only seed the bridge binds
and announces for it. The same hello confirms liveness, so no separate probe exists. At cold start the seed presented comes from the other realm's seed list.

**The translation rule has three lines.** A hash confirmed native in the receiving realm drops: that realm has the
peer, and a second copy only confuses its rosters. A bound native address becomes its translated seed. Any other
seed drops. A required field that drops refuses the request as an unavailable peer, HTTP 503; an optional field or
an answer list just loses the seed. An entry whose native address is unconfirmed answers the same way.

**Each forwarding endpoint owns its rules and its refusal answers; there is no rule engine.** The endpoint of a
path reads the configuration of its own crossing, for example refuse `transferRWI` into one realm with
`result=not_granted` and a pause, and answers in its path's vocabulary. The mux mounts only the paths `yacyproto`
names. A foreign network gets the hello answer YaCy gives, `yourtype=virgin`, and HTTP 403 elsewhere. Only the fields
that carry seeds and `yourip` are re-encoded: hello `seed`, search `myseed`, hello answer `seed0..N`. All else crosses byte for byte.

**Durable state is one vault with two collections: known addresses per realm, and the peer address table.** At start each
realm runs `RunOnce` to reconfirm its known addresses, then the listeners and the announcement cycle start. An entry ends when its native address stays unreachable past the retention, or a finite pool fills with departed peers.

## Residual risks and open points

* A rogue peer that claims another peer's hash reaches the other realm under it while the real peer is unconfirmed there.
* A dead door stays in `IPs` until every peer has tried it once; the bridge repeats gossip and cannot speed that up.
* The seed codec carries only the columns `yacymodel` names; a column it does not name does not cross.
* One realm's outbound client must leave through that realm: a bind address or an egress proxy per realm.
* ADRs to write before code: record decisions, the embedded storage engine, and the lifted wire libraries.
