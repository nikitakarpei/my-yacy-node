# yacyrealmbridge — Solution design

Accepted record of step 7 of `doc/design-process.md`: boxes, boundary contracts, decisions with reasons. The decisions that shape the service are landed in `adr/`; the specification carries the requirements.

## Mechanism

The bridge is a YaCy peer of its own in each realm and learns that realm's peers the way any peer does: from
seed lists, from the peers that greet it, and from the seeds carried in answers. It back-pings each peer before it
holds it in its view of the realm, keeps asking on a schedule, and drops a peer that stops answering. Each held
native peer gets one translated address in the other realm, and the bridge's hello answers in a realm list the
other realm's held peers at their translated addresses. Peers greet those addresses and gossip spreads the rest.

Every seed the bridge emits carries a bridge label in its `Tags`, signed with the bridge's key: identity seeds a
bare label, translated seeds a label naming the native address behind them. A bridge reads only labels signed by
keys its operator trusts; any other label is no label. A labelled seed is never a native peer, so no bridge translates
it onward, and no bridge translates a native peer a trusted label in the receiving realm names. Bridges never talk
to each other. When two translated one peer before seeing each other, the smaller translated address stands.

A request to a translated address is forwarded to the native address, and the answer comes back with every carried
seed translated; in a hello answer `yourip` becomes the address the bridge saw the caller on. A freeworld peer's
translated address in yggdrasil embeds its IPv4 and port inside the bridge's `/64`, so it needs no allocation. A
yggdrasil peer's translated address in freeworld is a port of the bridge's IPv4, allocated once. Instances of one
bridge share the views and the pool through NATS JetStream; how traffic reaches an instance is deployment.

## Units

| Package | Owns | Contract |
|---|---|---|
| `addressrealm` | The two realm names and a crossing between them. | `Name`, `Pair.OtherOf(Name) Name`, `Crossing{From, Into}` |
| `bridgelabel` | The bridge label a seed carries in its tags, signed with the bridge's key: the identity form, the translation form that names a native address, and the reading of a label signed by one of the configured trusted keys. | `IdentityLabel() []Tag`, `TranslationLabelOf(native) []Tag`, `LabelOf(seed) Optional[Label]`, `Label{Kind, NativeAddress, Signer}` |
| `bridgeidentity` | The bridge's own peer in one realm: its labelled seed with every capability flag off, and the endpoints a YaCy peer expects: hello, which hands the caller to the view and lists the other realm's peers; `query.html`, which answers rwicount 0; `search.html`, which answers nothing. | `Seed() yacymodel.Seed`, `NewMux(...) *http.ServeMux` |
| `realmpeerview` | What the bridge holds of one realm: the confirmed native peers with their seeds, and the translations other bridges' labels name there. Lives in a JetStream key-value bucket the bridge owns, watched into a live cache in each instance. Confirms by back-ping on arrival and on a schedule; drops after a configured run of failures. | `Offer(realm, seed)`, `NativeSeedAt(realm, address) Optional[Seed]`, `IsHeldNativeHash(realm, hash) bool`, `MostRecentlySeenSeeds(realm, limit) []Seed`, `TranslationsElsewhereOf(realm, native) []NetworkAddress` |
| `translationprecedence` | Which translation of one native peer stands in a realm: the bridge's own, or another bridge's at a smaller translated address. | `StandingTranslationOf(native, into) Optional[NetworkAddress]` |
| `addresstranslators/embedded` | The translated address of a native address as an IPv6 inside a configured `/64`: the IPv4 and port in the low 48 bits, one fixed port. Pure function both ways. | `TranslatedAddressOf(native) NetworkAddress`, `NativeAddressBehind(translated) Optional[NetworkAddress]` |
| `addresstranslators/pooled` | The translated address of a native address as a port of one configured host, allocated once by create-if-absent in a JetStream key-value bucket the bridge owns, and released when the peer drops from the view or its translation does not stand. | Same contract; `TranslatedAddressOf` reports `ok` false when the pool is full |
| `seedtranslation` | The seed one seed becomes when it enters a realm, and the rule: a labelled seed drops; a seed whose hash is held natively in the receiving realm drops; a held native peer whose own translation stands carries its translated address and label; any other seed drops. | `TranslatedSeedFor(seed, into) Optional[Seed]` |
| `peerrequestforwarding` | The `/yacy/` peer-protocol endpoints served on every translated address, one file per path. Each endpoint resolves the arriving address to a held native peer whose translation stands, applies the rules of its path for that crossing, refuses in place in its path's own vocabulary, or forwards with the seed fields and `yourip` translated. | `NewMux(...) *http.ServeMux` |
| `peerwire`, `peerbackping` | Wire units lifted from `yacynode` to `libraries/`: post one form to one peer address and read the message back; ask an address whether a YaCy peer of the network answers there. | `New(client, ...)` each |
| `<unit>observers/prometheus`, `<unit>observers/applog` | Metrics and logs of one unit, one observer interface per unit. | Per unit |
| `cmd/yacyrealmbridge` | Composition root: configuration, the JetStream connection, one outbound client per realm, the identity listener and the translated-address listeners per realm. | `RunService(ctx, cfg, registry)` |

## Decisions

**The bridge learns peers as any peer does, and an address is identity; a hash is payload.** A seed arrives from
the peer itself or from gossip, the back-ping is the one YaCy itself makes, and its schedule is the only liveness.
A peer that moves is confirmed at its new address and gets a new translated address; the old one drops when it
stops answering, and peers in the other realm drop it themselves. The hash serves one check: a hash held natively
in the receiving realm gets no translated address there.

**A signed label replaces coordination, and trust is configured.** Bridges over the same realms neither know nor
ask each other. A label under a trusted key is the whole signal, the native address in it is the key, and the
smaller translated address is the tie-break both sides compute alone. A bridge whose translation does not stand answers HTTP 503 on it, as
does a translated address behind which no held peer stands. A carried seed that does not translate: a required
field refuses the request the same way, an optional field or an answer list loses the seed.

**The translated address scheme is chosen per realm.** In yggdrasil every node owns a routed `/64` and answers on
any address in it, so a freeworld IPv4 and port embed in the low 48 bits: no state, and any instance decodes it.
Freeworld offers one IPv4, so a yggdrasil peer gets a port from a pool, allocated once. A translated seed sets `IP`
and `Port`, drops `IPs` and `PortSSL`, clears the SSL-available flag, and adds the label to `Tags`.

**A forwarded request leaves from the translated address of its caller when it enters yggdrasil.** A yggdrasil
peer compares the seed's address with the connecting address, and a lease rule there refuses a mismatch. Into
freeworld it leaves from the bridge's IPv4; YaCy tolerates the mismatch there and probes the advertised address.

**Each forwarding endpoint owns its rules and its refusal answers; there is no rule engine.** The endpoint of a
path reads the configuration of its own crossing, for example refuse `transferRWI` into one realm with
`result=not_granted` and a pause, and answers in its path's vocabulary. The mux mounts only the paths `yacyproto`
names. A foreign network gets `yourtype=virgin`, and HTTP 403 elsewhere. Only the fields that carry seeds and
`yourip` are re-encoded: hello `seed`, search `myseed`, hello answer `seed0..N`. All else crosses byte for byte.

**Shared state is JetStream buckets and nothing else.** Pebble and bbolt lock their directory to one process, so
no vault engine can serve two instances. The two views and the pool are flat maps, and create-if-absent gives the
one atomic step the pool needs. NATS is required, and a NATS cluster gives the bridge its HA.

## Residual risks and open points

* A seed whose label the bridge does not trust reads as a native peer: an untrusted bridge's translations can be translated onward, and two bridges that do not trust each other both translate every peer. Admission processes and trust chains between operators are future work. A hash claimed by a rogue is forwarded under it; not the bridge's concern.
* The yggdrasil docs call a `/64` unwise for identity verification, so a lease bound to a translated address there is only as strong as the `/64`; and one key is one node, so that `/64` lives on one router host in front of the instances, a SPOF the bridge cannot remove.
* Any peer that answers a back-ping is held, and one yggdrasil key answers on 2^64 addresses. The view and the pool are bounded, and those bounds are the whole defence.
* Verify on a host that the `/64` answers on addresses added to the interface before the embedded translator is written.
