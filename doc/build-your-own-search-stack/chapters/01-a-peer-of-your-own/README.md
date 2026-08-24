# 1. A peer of your own

> "I have a spare box in the corner. Can it be part of something bigger?"

Two containers. `yacy-rwi-node` joins the YaCy network, stores the postings
other peers hand it, and answers their searches. `smokescreen` sits in front of
every outbound connection and refuses the private ones, so nothing the node
talks to can walk into your LAN.

Distribution is on, so a posting the node holds moves along to the peers the
DHT makes responsible for it. With it off, everything the node collects stays
on your disk.

## Setup

```sh
cp .env.example .env
```

Set `YACY_ADVERTISE_HOST` to the address other peers reach you at, and open port
8090 to it. A peer nobody can reach still runs, but it collects nothing.

```sh
docker compose up -d
```

## Try it

```sh
docker compose logs -f yacy-rwi-node
curl -s localhost:9090/metrics
```

The first announce takes a few minutes. Peers arrive in the roster before any
posting does; a new peer is at the far end of everyone's list for a while.

## Where you are now

You are storing other people's pages and serving other people's searches.
Nothing in the index is yours yet.

> "How do I put pages I care about into it?"
