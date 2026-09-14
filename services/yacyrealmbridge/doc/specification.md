# yacyrealmbridge — Technical Specification

## Context

YaCy peers in different IP address realms can belong to one logical network. The bridge makes
the peers of each realm reachable from the other realm.

An address realm is an operator-defined IP reachability context. The bridge does not identify or
manage the network technology that provides it.

## Non-Goals

* Create, configure, monitor, or route an address realm.
* Provide a general-purpose proxy.
* Proxy non-peer YaCy interfaces.
* Provide anonymity or payload confidentiality from the bridge operator.
* Prove that a peer owns the peer hash it states.
* Establish trust between bridge operators.
* Talk to other bridges.

## Functional Requirements

* The bridge SHALL connect exactly two configured address realms.
* The bridge SHALL serve one configured YaCy network across both realms.
* The bridge SHALL be a YaCy peer of its own in each realm, SHALL learn that realm's peers the way any peer does, and SHALL publish in each realm a seed list that holds its own seed for that realm.
* Before it holds a peer, the bridge SHALL confirm that a YaCy peer of the network answers at the address the peer's seed advertises. The bridge SHALL keep confirming held peers and SHALL drop a peer that stops answering.
* The bridge SHALL lease each held peer hash one translated address in the other realm, from the translated address space the operator configures for that realm, for an operator-configured lease that the bridge renews while the peer is held. Until the lease expires, that hash SHALL have that translated address and no other.
* The bridge SHALL forward a translated address to the native address its view holds for the leased hash.
* The bridge SHALL NOT translate a peer when no translated address is available.
* The bridge SHALL mark its own seed in each realm with the bridge mark in the seed's tags, and SHALL NOT translate any seed that carries the bridge mark, whoever set it.
* The bridge SHALL mark every translated seed with a translation mark in the seed's tags, signed with the bridge's key over the seed it marks. The bridge SHALL read only a translation mark signed by a key the operator trusts, and SHALL treat a seed with any other translation mark as unmarked.
* The bridge SHALL NOT translate a seed that carries a trusted translation mark, and SHALL NOT translate a peer whose hash is held natively in the receiving realm.
* The bridge SHALL NOT translate a native peer whose hash a seed with a trusted translation mark in the receiving realm already carries. When two bridges have translated one hash, the translation with the smaller translated address SHALL stand, and the other bridge SHALL withdraw its translation and SHALL stop answering on it.
* The bridge SHALL list the held peers of one realm, at their translated addresses, in the hello answers of its peer in the other realm.
* The bridge SHALL preserve the peer hash, network, and DHT position of a held peer.
* The bridge SHALL change only the seed fields that describe how to reach the peer, and the tags. A translated seed SHALL offer plain HTTP at the translated address only.
* The bridge SHALL forward admitted YaCy peer requests that arrive at a standing translated address, and SHALL NOT answer on one that does not stand.
* The bridge SHALL translate every carried seed for the receiving realm. It SHALL refuse a request whose required seed has no translation, and SHALL drop an optional seed that has no translation.
* The bridge SHALL proxy only YaCy peer-protocol paths under `/yacy/`.
* The operator SHALL admit or refuse each peer-protocol path per crossing direction.
* The bridge SHALL answer a refused request in place of the native peer and SHALL NOT forward it.
* The bridge SHALL reject malformed seeds and requests for another YaCy network.

## Non-Functional Requirements

* A translated address SHALL stay with its hash until the lease expires, across bridge restarts and across moves of the native peer inside its realm.
* The bridge SHALL hold its views of both realms and its translated address leases in NATS JetStream, shared by every instance of the bridge, and SHALL hold no other durable state.
* Every instance of one bridge SHALL serve every held peer of that bridge.
* The bridge SHALL NOT start without NATS JetStream.
* The size of each view and of each translated address space SHALL have an operator-configured bound.
* Resource use and operation deadlines SHALL have operator-configured limits.
* Forwarded requests SHALL reach only held native addresses.
* Discovery failure in one realm SHALL NOT stop forwarding for the peers held in the other realm.
* Health, discovery, translation, forwarding, and refusal SHALL expose metrics.
* The bridge SHALL support low-resource Linux-class hosts.
* The bridge SHALL remain compatible with standard plain-HTTP YaCy peer contracts.

## Known Limitations

* A seed whose translation mark the bridge does not trust reads as a native peer. An untrusted bridge's translations can be translated onward, and two bridges that do not trust each other both translate every peer.
* A peer hash is stated, not proven. Peers in a realm that treat an address as identity see the bridge's own addresses behind every translated address, not the native peer.
* Any peer that answers a confirmation is held. The bounds on the view and on the address space are the whole defence.
* A rogue that states another peer's hash in one realm is forwarded under that hash into the other realm, as it would be inside one realm.
* The bridge's addresses in a realm may all sit on one host that the realm's technology binds to one key. That host is a single point of failure the bridge does not remove.
