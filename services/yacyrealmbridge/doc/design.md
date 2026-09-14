# yacyrealmbridge — Solution design

Accepted record of step 7 of `doc/design-process.md`: boxes, boundary contracts, decisions with reasons. The decisions that shape the service are landed in `adr/`; the specification carries the requirements.

## Mechanism

The bridge is a YaCy peer of its own in each realm, plus one translated address per registered peer in the
other realm. A peer chooses the bridge by adding the bridge's seed list, which holds only the bridge's seed in that
realm, and greeting it. The hello request carries the peer's seed first-hand; the bridge back-pings the address the
seed advertises and, when a YaCy peer answers there, registers the peer. Every later greeting renews a lease, and a
lease that lapses withdraws the peer. The bridge forwards only registered peers, and it never crawls a realm.

The bridge's hello answer lists, as known seeds, the registered peers of the other realm with their translated
addresses. Peers greet those addresses themselves, the bridge forwards each request to the native address, and
the answer comes back with every carried seed translated; in a hello answer `yourip` becomes the address the bridge
saw the caller on. Gossip spreads the rest. Anyone may run a bridge, and a peer is forwarded by the bridges it chose.

Instances of one bridge share their state through NATS JetStream and sit behind the bridge's address in each realm;
how traffic for a translated address reaches an instance is deployment. A freeworld peer's translated address in
yggdrasil embeds its IPv4 and port inside the bridge's `/64`, so it needs no allocation and any instance can decode
it. A yggdrasil peer's translated address in freeworld is a port of the bridge's IPv4, allocated once in a bucket.

## Units

| Package | Owns | Contract |
|---|---|---|
| `addressrealm` | The two realm names and a crossing between them. | `Name`, `Pair.OtherOf(Name) Name`, `Crossing{From, Into}` |
| `bridgeidentity` | The bridge's own peer in one realm: its seed with every capability flag off, and the endpoints a YaCy peer expects: hello, which registers the caller and lists the other realm's peers; `query.html`, which answers rwicount 0; `search.html`, which answers nothing. | `Seed() yacymodel.Seed`, `NewMux(...) *http.ServeMux` |
| `peerregistrations` | The registered native peers of both realms: native address, the seed the peer stated, and the lease. Lives in a JetStream key-value bucket the bridge owns, watched into a live cache in each instance. | `Register(realm, address, seed)`, `RegisteredSeedAt(realm, address) Optional[Seed]`, `IsRegisteredHash(realm, hash) bool`, `MostRecentlyRenewedSeeds(realm, limit) []Seed` |
| `addresstranslators/embedded` | The translated address of a native address as an IPv6 inside a configured `/64`: the IPv4 and port in the low 48 bits, one fixed port. Pure function both ways. | `TranslatedAddressOf(native) NetworkAddress`, `NativeAddressBehind(translated) Optional[NetworkAddress]` |
| `addresstranslators/pooled` | The translated address of a native address as a port of one configured host, allocated once by create-if-absent in a JetStream key-value bucket the bridge owns, and released when the registration lapses. | Same contract; `TranslatedAddressOf` reports `ok` false when the pool is full |
| `seedtranslation` | The seed one seed becomes when it enters a realm, and the rule: a seed of the bridge itself, or whose hash is registered natively in the receiving realm, drops; a seed of a peer registered in the sending realm carries its translated address; any other seed drops. | `TranslatedSeedFor(seed, into) Optional[Seed]` |
| `peerrequestforwarding` | The `/yacy/` peer-protocol endpoints served on every translated address, one file per path. Each endpoint resolves the arriving address to a registered native peer, applies the rules of its path for that crossing, refuses in place in its path's own vocabulary, or forwards with the seed fields and `yourip` translated. | `NewMux(...) *http.ServeMux` |
| `peerwire`, `peerbackping` | Wire units lifted from `yacynode` to `libraries/`: post one form to one peer address and read the message back; ask a caller's advertised address whether a YaCy peer of the network answers there. | `New(client, ...)` each |
| `<unit>observers/prometheus`, `<unit>observers/applog` | Metrics and logs of one unit, one observer interface per unit. | Per unit |
| `cmd/yacyrealmbridge` | Composition root: configuration, the JetStream connection, one outbound client per realm, the identity listener and the translated-address listeners per realm. | `RunService(ctx, cfg, registry)` |

## Decisions

**A peer registers by greeting the bridge; the bridge discovers nothing.** The seed in a hello request is stated
by the peer over its own connection, so the bridge repeats no rumour. The back-ping is the one YaCy itself makes.
The lease is the only liveness: no probe cycle, no announcement cycle. A peer that stops greeting is withdrawn
after the lease, and peers in the other realm drop its translated address themselves once it stops answering.

**An address is identity; a hash is payload.** A peer that moves registers again from its new address and gets a
new translated address; the old one lapses with its lease. The hash serves two checks only: a hash registered
natively in the receiving realm gets no translated address there, and the bridge's own seeds never cross.

**The translated address scheme is chosen per realm.** In yggdrasil every node owns a routed `/64` and answers on
any address in it, so a freeworld IPv4 and port embed in the low 48 bits: no state, stable by construction, and any
instance decodes it. Freeworld offers one IPv4, so a yggdrasil peer gets a port from a pool, allocated once. A
translated seed sets `IP` and `Port`, drops `IPs` and `PortSSL`, and clears the SSL-available flag: plain HTTP only.

**A forwarded request leaves from the translated address of its caller when it enters yggdrasil.** A yggdrasil
peer compares the seed's address with the connecting address, and a lease rule there refuses a mismatch. Into
freeworld it leaves from the bridge's IPv4; YaCy tolerates the mismatch there and probes the advertised address.

**Each forwarding endpoint owns its rules and its refusal answers; there is no rule engine.** The endpoint of a
path reads the configuration of its own crossing, for example refuse `transferRWI` into one realm with
`result=not_granted` and a pause, and answers in its path's vocabulary. The mux mounts only the paths `yacyproto`
names. A foreign network gets `yourtype=virgin`, and HTTP 403 elsewhere. Only the fields that carry seeds and
`yourip` are re-encoded: hello `seed`, search `myseed`, hello answer `seed0..N`. All else crosses byte for byte.

**A carried seed that does not translate: required field refuses the request as an unavailable peer, HTTP 503;
optional field or answer list loses the seed.** A translated address with no live registration answers the same way.

**Shared state is two JetStream buckets and nothing else.** Pebble and bbolt lock their directory to one process,
so no vault engine can serve two instances. Registrations and pooled allocations are two flat maps; a bucket with
create-if-absent gives the one atomic step the pool needs. NATS is required, and a NATS cluster gives the bridge its HA.

## Residual risks and open points

* The yggdrasil docs call a `/64` unwise for identity verification: a lease bound to a translated address there is only as strong as the `/64`.
* One yggdrasil key is one node, so the bridge's `/64` lives on one router host in front of the instances. That host is a SPOF the bridge cannot remove.
* Anyone in a realm may register and hold a translated address until the lease lapses; the pool is bounded, and that bound is the whole defence.
* A hash claimed by a rogue in one realm is forwarded under it into the other while the real peer is unregistered there. Not the bridge's concern.
* The seed codec carries only the columns `yacymodel` names.
* Verify on a host that the `/64` answers on addresses added to the interface before the embedded translator is written.
