# yacyrealmbridge — Solution design

Accepted record of step 7 of `doc/design-process.md`: boxes, boundary contracts, decisions with reasons. The decisions that shape the service are landed in `adr/`; the specification carries the requirements.

## Mechanism

The bridge is a YaCy peer of its own in each realm and learns that realm's peers the way any peer does: from
seed lists, from the peers that greet it, and from the seeds carried in answers. It back-pings each peer before it
holds it in its view of the realm, keeps asking on a schedule, and drops a peer that stops answering. Each held
peer hash leases one translated address in the other realm, and the bridge's hello answers in a realm list the
other realm's held peers at their translated addresses. Peers greet those addresses and gossip spreads the rest.

The bridge's own seed carries the bridge mark in its `Tags`, and a public bridge, the default, puts a translation
mark signed with its key on every translated seed. No bridge translates a seed with either mark, verified or not:
a mark costs only whoever set it, and a peer crosses one public bridge, never a chain. What a translation mark
states counts only under a key the operator trusts: a hash a trusted translation in the receiving realm carries is
not translated again. A private bridge, for a lone deployment, marks no translated seed, so they read as native.

A request to a translated address is forwarded to the native address the view holds for the leased hash, and the
answer comes back with every carried seed translated; in a hello answer `yourip` becomes the address the bridge saw
the caller on. Each realm has a configured translated address space: in yggdrasil the addresses of the bridge's
`/64`, in freeworld the ports of the bridge's IPv4. A lease takes one address once per hash and frees it at expiry.
Instances of one bridge share the views and the leases through NATS JetStream; how traffic reaches one is deployment.

## Units

| Package | Owns | Contract |
|---|---|---|
| `addressrealm` | The two realm names and a crossing between them. | `Name`, `Pair.OtherOf(Name) Name`, `Crossing{From, Into}` |
| `bridgemark` | The plain tag a bridge's own seed carries, and whether a seed carries it. | `Tag() Tag`, `IsBridgeSeed(seed) bool` |
| `translationmark` | The tags a public bridge's translated seed carries, signed with the bridge's key over the seed; whether a seed carries any; and the reading of a mark signed by one of the configured trusted keys. | `TagsFor(seed) []Tag`, `IsMarked(seed) bool`, `TrustedMarkOf(seed) Optional[Mark]`, `Mark{Signer}` |
| `bridgeidentity` | The bridge's own peer in one realm: its marked seed with every capability flag off, and the endpoints a YaCy peer expects: hello, which hands the caller to the view and lists the other realm's peers; `query.html`, which answers rwicount 0; `search.html`, which answers nothing. | `Seed() yacymodel.Seed`, `NewMux(...) *http.ServeMux` |
| `realmpeerview` | What the bridge holds of one realm: the confirmed unmarked peers and trusted translations by hash with their seeds, and the translated addresses trusted marks carry there by hash. Lives in a JetStream key-value bucket the bridge owns, watched into a live cache in each instance. Confirms by back-ping on arrival and on a schedule; drops after a configured run of failures. | `Offer(realm, seed)`, `HeldSeedOf(realm, hash) Optional[Seed]`, `MostRecentlySeenSeeds(realm, limit) []Seed`, `TranslationsElsewhereOf(realm, hash) []NetworkAddress` |
| `translatedaddressleases` | The translated address one hash leases in one realm: taken once from that realm's address space by create-if-absent in a JetStream key-value bucket the bridge owns, renewed while the hash is held, freed at expiry. | `LeasedAddressFor(hash, into) Optional[NetworkAddress]`, `HashBehind(translated) Optional[PeerHash]`, `Renew(hash, into)`; `ok` false when the space is full |
| `translatedaddressspaces/prefixaddresses`, `translatedaddressspaces/hostports` | The addresses a realm's space offers: every address of one configured prefix at one fixed port, or every port of one configured host. | `FreeAddressAmong(taken) Optional[NetworkAddress]` |
| `translationprecedence` | Which translation of one hash stands in a realm: the bridge's own lease, or another bridge's at a smaller translated address. | `StandingTranslationOf(hash, into) Optional[NetworkAddress]` |
| `seedtranslation` | The seed one seed becomes when it enters a realm, and the rule: a seed with the bridge mark or any translation mark drops; a held peer whose own translation stands carries its leased address and, from a public bridge, a translation mark; any other seed drops. | `TranslatedSeedFor(seed, into) Optional[Seed]` |
| `peerrequestforwarding` | The `/yacy/` peer-protocol endpoints served on every translated address, one file per path. Each endpoint resolves the arriving address to a leased hash whose translation stands and to the seed the view holds for it, applies the rules of its path for that crossing, refuses in place in its path's own vocabulary, or forwards with the seed fields and `yourip` translated. | `NewMux(...) *http.ServeMux` |
| `peerwire`, `peerbackping` | Wire units lifted from `yacynode` to `libraries/`: post one form to one peer address and read the message back; ask an address whether a YaCy peer of the network answers there. | `New(client, ...)` each |
| `<unit>observers/prometheus`, `<unit>observers/applog` | Metrics and logs of one unit, one observer interface per unit. | Per unit |
| `cmd/yacyrealmbridge` | Composition root: configuration, the JetStream connection, one outbound client per realm, the identity listener and the translated-address listeners per realm. | `RunService(ctx, cfg, registry)` |

## Decisions

**The bridge learns peers as any peer does, and a hash leases one translated address.** A seed arrives from the
peer itself or from gossip, the back-ping is the one YaCy itself makes, and its schedule is the only liveness of
the view. A hash keeps its translated address until the lease expires, whatever its native address does; the
native address behind it is what the view holds for the hash, as any roster does. A rogue stating another peer's
hash, or a native seed of the same hash in the receiving realm, is a conflict that realm's peers resolve themselves.

**A signed translation mark replaces coordination, and trust is configured.** Bridges over the same realms
neither know nor ask each other. A mark under a trusted key is the whole signal, the hash of the seed it marks is
the key, and the smaller translated address is the tie-break both sides compute alone. Any translation mark stops
a seed, so no chain forms; the signature is best effort against two trusted bridges repeating each other. A bridge whose translation does not stand
answers HTTP 503 on it, as does a translated address behind which no held peer stands. A carried seed that does not
translate: a required field refuses the request the same way, an optional field or an answer list loses the seed.

**Translated addresses are leased from a configured address space per realm.** In yggdrasil every node owns a
routed `/64` and answers on any address in it, so the space is that prefix; freeworld offers one IPv4, so the space
is its ports. Deriving the address from the native address was rejected: a move would change it. A translated seed
sets `IP` and `Port`, drops `IPs` and `PortSSL`, clears the SSL-available flag, and adds the translation mark to `Tags`.

**A forwarded request leaves from the translated address of its caller when it enters yggdrasil.** A yggdrasil
peer compares the seed's address with the connecting address, and a lease rule there refuses a mismatch. Into
freeworld it leaves from the bridge's IPv4; YaCy tolerates the mismatch there and probes the advertised address.

**Each forwarding endpoint owns its rules and its refusal answers; there is no rule engine.** The endpoint of a
path reads the configuration of its own crossing, for example refuse `transferRWI` into one realm with
`result=not_granted` and a pause, and answers in its path's vocabulary. The mux mounts only the paths `yacyproto`
names. A foreign network gets `yourtype=virgin`, and HTTP 403 elsewhere. Only the fields that carry seeds and
`yourip` are re-encoded: hello `seed`, search `myseed`, hello answer `seed0..N`. All else crosses byte for byte.

**Shared state is JetStream buckets and nothing else.** Pebble and bbolt lock their directory to one process, so
no vault engine can serve two instances. The two views and the leases are flat maps, and create-if-absent gives
the one atomic step a lease needs. NATS is required, and a NATS cluster gives the bridge its HA.

## Residual risks and open points

* The yggdrasil docs call a `/64` unwise for identity verification, so a lease bound to a translated address there is only as strong as the `/64`; and one key is one node, so that `/64` lives on one router host in front of the instances, a SPOF the bridge cannot remove. Verify on a host that the `/64` answers on addresses added to the interface before the prefix address space is written.
* Any peer that answers a back-ping is held, and one yggdrasil key answers on 2^64 addresses; the view and the port space are bounded, and those bounds are the whole defence. Two bridges that do not trust each other both translate every hash, and a private bridge's translations read as native to every other bridge, which may translate them again. Admission processes and trust chains between operators are future work.
