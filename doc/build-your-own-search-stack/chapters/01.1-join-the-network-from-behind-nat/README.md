# 1.1 Join the network from behind NAT

> "Can I run a peer without a public IP or a rented server?"

Chapter 1 needs an address that other peers reach on port 8090. NAT,
carrier-grade NAT, and changing addresses hide it. A tunnel grants a public
hostname instead.

## What this chapter adds

- `localhostrunagent` opens a tunnel to localhost.run and passes its traffic to
  the node. It leases the public hostname to the node, and the node announces
  that hostname to other peers.

Port 8090 stays private, and the lease replaces `YACY_ADVERTISE_HOST`.

Optionally
[register an SSH key](../../../../services/localhostrunagent/doc/configuration.md)
to keep one hostname for longer. Each new hostname restarts the node.

## Start

Make a peer key once. The peer hash and the data of the node belong to this key, so
keep `.env` safe:

```sh
echo "YACY_PEER_KEY=$(docker run --rm ghcr.io/nikitakarpei/my-yacy-node/yacynode:latest yacy-peer-key)" >> .env
```

```sh
docker compose up -d
```

## Use

Read the granted hostname and the peer hash, then ask for the posting count:

```sh
docker compose logs localhostrunagent | grep 'localhostrunagent started'
docker compose logs yacy-rwi-node | grep 'node identity'
curl -fsS 'http://<hostname>/yacy/query.html?object=rwicount&youare=<peer>&iam=AAAAAAAAAAAA'
```

A `response` line confirms that peers reach the node through the tunnel.

## More information

- [Tunnel configuration](../../../../services/localhostrunagent/doc/configuration.md)
- [Tunnel behavior](../../../../services/localhostrunagent/doc/specification.md)
- [Node configuration](../../../../services/yacynode/doc/configuration.md)
