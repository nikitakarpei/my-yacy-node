# What this is

[![ci](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml)
[![go](https://img.shields.io/github/go-mod/go-version/nikitakarpei/my-yacy-node?filename=services%2Fyacynode%2Fgo.mod)](https://github.com/nikitakarpei/my-yacy-node/blob/main/services/yacynode/go.mod)
[![license](https://img.shields.io/github/license/nikitakarpei/my-yacy-node)](LICENSE)
[![images](https://img.shields.io/badge/ghcr.io-amd64%20%7C%20arm64-blue)](https://github.com/nikitakarpei?tab=packages)

A lightweight reimplementation of a [YaCy](https://yacy.net) peer, built as
small, single-purpose Go services.

> “YaCy is a distributed Web Search Engine, based on a peer-to-peer network.”
> — [YaCy FAQ](https://yacy.net/faq/)

## Why it exists

I, the author of this project, got very passionate when I learned about the
existence of a peer-to-peer search engine, but was left unsatisfied when I tried
it, due to several problems such as high memory consumption, poor quality of
search results and security weaknesses of the protocol.

In the spring of 2026 I decided to leave my job to take a rest and finally spend
some time building what *I* want. So I started this experimental project with
the hope of making something that other people would find useful, and that would
make peer-to-peer search engines more popular.

## What is achieved so far

Compared to the Java implementation, this one:

- prevents peers from claiming the identity of another peer by tying each
  identity to a
  [locally generated private key](libraries/yacymodel/doc/adr/0001-derive-the-peer-hash-from-keys-behind-a-work-puzzle.md).
- enforces [HTTPS](services/yacynode/doc/adr/0024-serve-http-and-tls-on-one-peer-port.md)
  between peers running this implementation, keeping HTTP for compatibility
  with Java peers.
- prevents man-in-the-middle (MITM) attacks by accepting only TLS certificates
  that are [endorsed by the peer's key](libraries/yacymodel/doc/adr/0005-bind-tls-keys-to-the-peer-identity.md).
- improves relevance of search results by adopting new relevance signals and
  by training a [classifier](services/spamproxy/doc/specification.md) that
  hides spam and harmful pages.
- improves search time by tracking the response time of peers and by using
  heuristics like hedged requests and response header timeouts to switch to
  other peers faster.

## What is planned

Next, this one will:

- prove to other peers that the index holds what the origin served, unmodified,
  by building it on an
  [Android companion app](distributions/yacypeercompanion/README.md) that signs
  the fetched pages with hardware keys.
- make censorship harder by stopping peers from choosing their own position on
  the DHT ring.

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
