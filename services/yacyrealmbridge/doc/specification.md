# yacyrealmbridge — Technical Specification

## Context

YaCy peers in different IP address realms can belong to one logical network. The bridge makes
the peers of each realm reachable from the other realm. An address realm is an operator-defined
IP reachability context. The bridge does not identify or manage the network technology that
provides it.

## Non-Goals

* Create, configure, monitor, or route an address realm.
* Provide a general-purpose proxy, or proxy non-peer YaCy interfaces.
* Provide anonymity or payload confidentiality from the bridge operator.
* Prove that a peer owns the peer hash it states.
* Establish trust between bridge operators, or talk to other bridges.
* Carry a peer across more than one public bridge.

## Functional Requirements

* The bridge SHALL connect exactly two configured address realms.
* The bridge SHALL serve one configured YaCy network across both realms.
* The bridge SHALL be a YaCy peer in each realm and SHALL publish its seed there in a seed list.
* The bridge SHALL learn each realm's peers from seed lists, greetings, and seeds in answers.
* The bridge SHALL hold a peer only after a YaCy peer of the network has answered at the address
  the peer's seed advertises, and SHALL drop a held peer that stops answering.
* The bridge SHALL give each held peer hash one translated address in the other realm, and no
  other, until an operator-configured lease that renews while the peer is held expires.
* The operator SHALL configure the translated address space of each realm.
* The bridge SHALL NOT translate a peer when no translated address is available.
* The bridge SHALL mark its own seed in each realm as a bridge seed, and SHALL NOT translate a
  seed marked as a bridge seed, whoever marked it.
* A public bridge, the default, SHALL mark every translated seed as a translation, signed with
  its key; a private bridge SHALL NOT mark translated seeds.
* The bridge SHALL NOT translate a seed that carries a translation mark, verified or not.
* The bridge SHALL honour what a translation mark states only under a key the operator trusts.
* The bridge SHALL translate a peer even when the receiving realm holds a native seed of its hash.
* The bridge SHALL NOT translate a peer whose hash a trusted translation in the receiving realm
  already carries.
* When two bridges have translated one hash, the translation whose mark states the earlier lease
  SHALL stand, the smaller address on equal times, and the other bridge SHALL withdraw it.
* The bridge SHALL list the held peers of one realm, at their translated addresses, in the hello
  answers of its peer in the other realm.
* The bridge SHALL preserve the peer hash, network, and DHT position of a held peer.
* The bridge SHALL change only the seed fields that describe how to reach the peer, and the tags.
* A translated seed SHALL offer plain HTTP at the translated address only.
* The bridge SHALL forward admitted peer requests that arrive at a standing translated address to
  the native address it holds for that hash, and SHALL NOT answer on one that does not stand.
* The bridge SHALL translate every carried seed for the receiving realm.
* The bridge SHALL refuse a request whose required seed has no translation, and SHALL drop an
  optional seed that has no translation.
* The bridge SHALL proxy only YaCy peer-protocol paths under `/yacy/`.
* The operator SHALL admit or refuse each peer-protocol path per crossing direction.
* The bridge SHALL answer a refused request in place of the native peer and SHALL NOT forward it.
* The bridge SHALL reject malformed seeds and requests for another YaCy network.

## Non-Functional Requirements

* A translated address SHALL survive bridge restarts and moves of the native peer inside its realm.
* The bridge SHALL hold no durable state beyond its views of the realms and its leases.
* Every instance of one bridge SHALL serve every held peer of that bridge.
* The bridge SHALL NOT start without NATS JetStream.
* Each view, each translated address space, resource use, and operation deadlines SHALL have
  operator-configured limits.
* Discovery failure in one realm SHALL NOT stop forwarding for the peers held in the other realm.
* Health, discovery, translation, forwarding, and refusal SHALL expose metrics.
* The bridge SHALL support low-resource Linux-class hosts.
* The bridge SHALL remain compatible with standard plain-HTTP YaCy peer contracts.

## Known Limitations

* A translation mark under a key the bridge does not trust stops that seed and nothing else: two
  bridges that do not trust each other both translate every peer.
* A private bridge's translations look native to every other bridge, which may translate them again.
* A rogue that states another peer's hash is forwarded under it, or conflicts with the true peer
  in the receiving realm; the peers of that realm resolve the conflict by their own rules.
* Where an address is identity, a translated peer shows the bridge's address, not its own.
* Any peer that answers is held; the bounds on the view and the address space are the whole defence.
* The bridge's addresses in a realm may all sit on one host that the realm's technology binds to
  one key. That host is a single point of failure the bridge does not remove.
