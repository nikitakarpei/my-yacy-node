# YaCy Realm Bridge — Draft Specification

## Context

YaCy peers in different IP address realms can belong to one logical network.
The bridge makes peers in each realm reachable from the other realm.

An address realm is an operator-defined IP reachability context. The bridge does
not identify or manage the network technology that provides it.

## Non-Goals

* Create, configure, monitor, or route an address realm.
* Provide a general-purpose proxy.
* Change YaCy peer identities or DHT behavior.
* Proxy non-peer YaCy interfaces.
* Provide anonymity or payload confidentiality from the bridge operator.

## Functional Requirements

* The bridge SHALL connect exactly two configured address realms.
* The bridge SHALL serve one configured YaCy network across both realms.
* The bridge SHALL discover and confirm peers independently in each realm.
* The bridge SHALL give each projected peer a unique endpoint in the other realm.
* The bridge SHALL preserve the peer hash, network, capabilities, and DHT position.
* The bridge SHALL replace only seed fields that describe reachability.
* The bridge SHALL announce projected seeds in the receiving realm.
* The bridge SHALL forward YaCy peer requests between projected and native endpoints.
* The bridge SHALL translate every carried seed for the receiving realm.
* The bridge SHALL proxy only configured YaCy peer-protocol paths under `/yacy/`.
* The bridge SHALL suppress a projection when that peer is native to the receiving realm.
* The bridge SHALL withdraw a projection while its native peer is unreachable.
* The bridge SHALL reject malformed seeds and requests for another YaCy network.
* The bridge SHALL refuse new projections when no endpoint is available.

## Non-Functional Requirements

* Projection endpoints SHALL remain stable across peer moves and bridge restarts.
* The bridge SHALL restore valid projections before it resumes announcements.
* Resource use and operation deadlines SHALL have operator-configured limits.
* Forwarded requests SHALL reach only confirmed native peer endpoints.
* Discovery failure in one realm SHALL not stop valid projections in the other realm.
* Health, projection, discovery, announcement, and forwarding SHALL expose metrics.
* The bridge SHALL support low-resource Linux-class hosts.
* The bridge SHALL remain compatible with standard plain-HTTP YaCy peer contracts.
