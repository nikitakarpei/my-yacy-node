# yacyrealmbridge — Technical Specification

## Context

YaCy peers in different IP address realms can belong to one logical network. The bridge makes
the peers that register with it in each realm reachable from the other realm.

An address realm is an operator-defined IP reachability context. The bridge does not identify or
manage the network technology that provides it.

## Non-Goals

* Create, configure, monitor, or route an address realm.
* Provide a general-purpose proxy.
* Proxy non-peer YaCy interfaces.
* Provide anonymity or payload confidentiality from the bridge operator.
* Find peers that did not register with the bridge.
* Prove that a peer owns the peer hash it states.

## Functional Requirements

* The bridge SHALL connect exactly two configured address realms.
* The bridge SHALL serve one configured YaCy network across both realms.
* The bridge SHALL be a YaCy peer of its own in each realm, and SHALL publish in each realm a seed list that holds only its own seed for that realm.
* A peer SHALL register with the bridge when it greets the bridge's peer in its own realm. Before it registers the peer, the bridge SHALL confirm that a YaCy peer of the network answers at the address the greeting seed advertises.
* The bridge SHALL identify a registered peer by its native address. The peer hash SHALL be payload.
* The bridge SHALL hold a registration for an operator-configured lease. Each greeting SHALL renew the lease. The bridge SHALL withdraw a registration and its translated address when the lease lapses.
* The bridge SHALL give each registered peer one translated address in the other realm, by the translated address scheme the operator configures for that realm.
* The bridge SHALL refuse a registration when no translated address is available in the other realm.
* The bridge SHALL give no translated address in a realm to a peer whose hash is registered natively in that realm, and SHALL NOT carry its own seeds across realms.
* The bridge SHALL list the registered peers of one realm, at their translated addresses, in the hello answers of its peer in the other realm.
* The bridge SHALL preserve the peer hash, network, and DHT position of a registered peer.
* The bridge SHALL change only the seed fields that describe how to reach the peer. A translated seed SHALL offer plain HTTP at the translated address only.
* The bridge SHALL forward admitted YaCy peer requests that arrive at a translated address to the native address of the registered peer.
* The bridge SHALL translate every carried seed for the receiving realm. It SHALL refuse a request whose required seed has no translation, and SHALL drop an optional seed that has no translation.
* The bridge SHALL proxy only YaCy peer-protocol paths under `/yacy/`.
* The operator SHALL admit or refuse each peer-protocol path per crossing direction.
* The bridge SHALL answer a refused request in place of the native peer and SHALL NOT forward it.
* The bridge SHALL reject malformed seeds and requests for another YaCy network.

## Non-Functional Requirements

* A translated address SHALL stay the same while the native address behind it holds its registration, across bridge restarts.
* The bridge SHALL hold registrations and translated address allocations in NATS JetStream, shared by every instance of the bridge, and SHALL hold no other durable state.
* Every instance of one bridge SHALL serve every registered peer of that bridge.
* The bridge SHALL NOT start without NATS JetStream.
* Resource use and operation deadlines SHALL have operator-configured limits.
* Forwarded requests SHALL reach only registered native addresses.
* Registration failure in one realm SHALL NOT stop forwarding for the peers registered in the other realm.
* Health, registration, translation, forwarding, and refusal SHALL expose metrics.
* The bridge SHALL support low-resource Linux-class hosts.
* The bridge SHALL remain compatible with standard plain-HTTP YaCy peer contracts.

## Known Limitations

* A registration is as strong as the native address the realm gives the peer. Peers in a realm that treat an address as identity see the bridge's own addresses behind every translated address, not the native peer.
* Anyone in a realm may register and hold a translated address until the lease lapses. The number of translated addresses in a realm is bounded, and that bound is the whole defence.
* A hash that a rogue states in one realm is forwarded under that hash into the other realm while the real peer is not registered there.
* The bridge's addresses in a realm may all sit on one host that the realm's technology binds to one key. That host is a single point of failure the bridge does not remove.
* A peer is reachable in the other realm only through the bridges it registered with. Anyone may run a bridge.
