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
* The bridge SHALL be a YaCy peer in each realm.
* The bridge SHALL learn each realm's peers from seed lists, greetings, and seeds in answers.
* The bridge SHALL confirm a peer only after a YaCy peer of the network answers at the address
  in the peer's seed, and SHALL drop a confirmed peer that stops answering.
* The bridge SHALL give each confirmed peer one translated address in the other realm, and SHALL
  keep that address for the peer's hash until an operator-configured lease expires.
* The lease SHALL renew while the peer stays confirmed.
* The operator SHALL configure the translated address space of each realm.
* The bridge SHALL NOT translate a peer when no translated address is free.
* The bridge SHALL mark its own seed in each realm as a bridge seed, and SHALL NOT translate any
  seed with a bridge mark, whoever set it.
* A public bridge, the default, SHALL put on every seed it translates a translation mark that
  shows when the lease began, signed with its key. A private bridge SHALL NOT mark seeds.
* The bridge SHALL NOT translate any seed with a translation mark, verifiable or not.
* The bridge SHALL read a translation mark only if a key the operator trusts signed it.
* The bridge SHALL translate a peer even when the receiving realm already has a native seed with
  the same hash.
* The bridge SHALL NOT translate a peer whose hash is already in a trusted translation in the
  receiving realm.
* When two bridges have translated the same hash, the one whose mark shows the earlier time
  SHALL stay, the smaller address if the times are equal, and the other bridge SHALL remove its own.
* The bridge SHALL list the confirmed peers of one realm, at their translated addresses, in the
  hello answers it gives in the other realm.
* A translated seed SHALL keep the peer hash and the network of the peer.
* A translated seed SHALL differ from the peer's own seed only in the fields that say how to
  reach the peer, and in the tags.
* The bridge SHALL forward an admitted request that arrives at a translated address to the peer
  behind that address, and SHALL translate every seed in the request and the answer.
* The bridge SHALL proxy only YaCy peer-protocol paths under `/yacy/`.
* The operator SHALL admit or refuse each peer-protocol path for each direction of crossing.

## Non-Functional Requirements

* A translated address SHALL survive bridge restarts and moves of the native peer inside its realm.
* The bridge SHALL keep no durable state beyond its views of the realms and its leases.
* Every instance of one bridge SHALL serve every peer that bridge confirmed.
* The bridge SHALL NOT start without NATS JetStream.
* Each view, each translated address space, resource use, and operation deadlines SHALL have
  operator-configured limits.
* Discovery failure in one realm SHALL NOT stop forwarding for the peers confirmed in the other.
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
* Any peer that answers is confirmed; the view and address space bounds are the whole defence.
* The bridge's addresses in a realm may all sit on one host that the realm's technology binds to
  one key. That host is a single point of failure the bridge does not remove.
