# What this is

[![ci](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml)
[![go](https://img.shields.io/github/go-mod/go-version/nikitakarpei/my-yacy-node?filename=services%2Fyacynode%2Fgo.mod)](https://github.com/nikitakarpei/my-yacy-node/blob/main/services/yacynode/go.mod)
[![license](https://img.shields.io/github/license/nikitakarpei/my-yacy-node)](LICENSE)
[![images](https://img.shields.io/badge/ghcr.io-amd64%20%7C%20arm64-blue)](https://github.com/nikitakarpei?tab=packages)
[![openssf scorecard](https://api.scorecard.dev/projects/github.com/nikitakarpei/my-yacy-node/badge)](https://scorecard.dev/viewer/?uri=github.com/nikitakarpei/my-yacy-node)

A lightweight reimplementation of a [YaCy](https://yacy.net) peer, built as
small, single-purpose Go services.

> “YaCy is a distributed Web Search Engine, based on a peer-to-peer network.”
> — [YaCy FAQ](https://yacy.net/faq/)

## Why it exists

I learned about YaCy from some Hacker News thread and got very passionate about
the idea of a peer-to-peer search engine. When I ran a peer myself, I found
issues with memory consumption, quality of search results and security of its
peer-to-peer protocol. So when I left my job and had some free time, I started
this experimental project to try to fix them, and to help make peer-to-peer
search useful to more people.

## What is achieved so far

Compared to the Java implementation, this one:

- stops peers from [claiming the identity of another peer][identity].
- requires encrypted traffic between its own peers and [binds TLS to the
  peer's key][tls], which blocks man-in-the-middle attacks.
- hides spam and harmful pages from search results with a [trained
  classifier][spam].
- answers searches faster by doing more work in parallel and waiting less
  for slow peers.

[identity]: libraries/yacymodel/doc/adr/0001-derive-the-peer-hash-from-keys-behind-a-work-puzzle.md
[tls]: libraries/yacymodel/doc/adr/0005-bind-tls-keys-to-the-peer-identity.md
[spam]: services/spamproxy/doc/specification.md

## What is planned

Next, this one will:

- prove to other peers that its index holds the pages as the origin served
  them, using pages signed by an [Android companion app][companion].
- make censorship harder by stopping peers from choosing their own position on
  the DHT ring.

[companion]: distributions/yacypeercompanion/README.md

## How to get started

> This project is under active development and has no stable release yet.
> Expect breaking changes. Some features are not available yet, and others work
> differently.

You can see it at work on
[search.my-yacy-node.org](https://search.my-yacy-node.org/). Feel free to play
around with it.

If you decide to run it yourself, I advise you to try the
[single-binary deployment](distributions/yacypeer/README.md) first.

Read [Build your own search stack](doc/build-your-own-search-stack/README.md) to
see how to compose the services into customized deployments.

## Documentation

Each service under `services/`, distribution under `distributions/`, and plugin
under `plugins/` has its own `doc/` directory:

- `specification.md` covers scope, requirements, and known limitations.
- `configuration.md` covers operator-facing settings.
