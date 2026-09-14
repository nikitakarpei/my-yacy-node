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
* Decide which bridge keys to trust.
* Talk to other bridges.

## Functional Requirements

* The bridge SHALL connect exactly two configured address realms.
* The bridge SHALL serve one configured YaCy network across both realms.
* The bridge SHALL be a YaCy peer of its own in each realm, SHALL learn that realm's peers the way any peer does, and SHALL publish in each realm a seed list that holds its own seed for that realm.
* Before it holds a peer, the bridge SHALL confirm that a YaCy peer of the network answers at the address the peer's seed advertises. The bridge SHALL keep confirming held peers and SHALL drop a peer that stops answering.
* The bridge SHALL identify a held peer by its native address. The peer hash SHALL be payload.
* The bridge SHALL give each held native peer one translated address in the other realm, by the translated address scheme the operator configures for that realm, and SHALL NOT translate a peer when no translated address is available.
* The bridge SHALL mark every seed it emits with a bridge label in the seed's tags, signed with the bridge's key. The label of a translated seed SHALL name the native address behind it. The bridge SHALL read only a label whose signature holds.
* The bridge SHALL NOT translate a labelled seed, and SHALL NOT translate a peer whose hash is held natively in the receiving realm.
* The bridge SHALL NOT translate a native peer whose native address a label in the receiving realm already names. When two bridges have translated one peer, the translation with the smaller translated address SHALL stand, and the other bridge SHALL withdraw its translation and SHALL stop answering on it.
* The bridge SHALL list the held peers of one realm, at their translated addresses, in the hello answers of its peer in the other realm.
* The bridge SHALL preserve the peer hash, network, and DHT position of a held peer.
* The bridge SHALL change only the seed fields that describe how to reach the peer, and the tags. A translated seed SHALL offer plain HTTP at the translated address only.
* The bridge SHALL forward admitted YaCy peer requests that arrive at a standing translated address to the native address of the held peer.
* The bridge SHALL translate every carried seed for the receiving realm. It SHALL refuse a request whose required seed has no translation, and SHALL drop an optional seed that has no translation.
* The bridge SHALL proxy only YaCy peer-protocol paths under `/yacy/`.
* The operator SHALL admit or refuse each peer-protocol path per crossing direction.
* The bridge SHALL answer a refused request in place of the native peer and SHALL NOT forward it.
* The bridge SHALL reject malformed seeds and requests for another YaCy network.

## Non-Functional Requirements

* A translated address SHALL stay the same while the native address behind it is held, across bridge restarts.
* The bridge SHALL hold its views of both realms and its translated address allocations in NATS JetStream, shared by every instance of the bridge, and SHALL hold no other durable state.
* Every instance of one bridge SHALL serve every held peer of that bridge.
* The bridge SHALL NOT start without NATS JetStream.
* The size of each view and of each translated address pool SHALL have an operator-configured bound.
* Resource use and operation deadlines SHALL have operator-configured limits.
* Forwarded requests SHALL reach only held native addresses.
* Discovery failure in one realm SHALL NOT stop forwarding for the peers held in the other realm.
* Health, discovery, translation, forwarding, and refusal SHALL expose metrics.
* The bridge SHALL support low-resource Linux-class hosts.
* The bridge SHALL remain compatible with standard plain-HTTP YaCy peer contracts.

## Known Limitations

* A bridge label is signed, but the bridge trusts every key. A rogue that gossips a seed labelled as some peer's translation, at a small address, stops every honest bridge from translating that peer.
* A held peer is as strong as the native address the realm gives it. Peers in a realm that treat an address as identity see the bridge's own addresses behind every translated address, not the native peer.
* Any peer that answers a confirmation is held. The bounds on the view and on the pool are the whole defence.
* A hash that a rogue states in one realm is forwarded under that hash into the other realm.
* The bridge's addresses in a realm may all sit on one host that the realm's technology binds to one key. That host is a single point of failure the bridge does not remove.
