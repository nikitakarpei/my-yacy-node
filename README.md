# What this is

[![ci](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml)
[![go](https://img.shields.io/github/go-mod/go-version/nikitakarpei/my-yacy-node?filename=services%2Fyacynode%2Fgo.mod)](https://github.com/nikitakarpei/my-yacy-node/blob/main/services/yacynode/go.mod)
[![license](https://img.shields.io/github/license/nikitakarpei/my-yacy-node)](LICENSE)
[![images](https://img.shields.io/badge/ghcr.io-amd64%20%7C%20arm64-blue)](https://github.com/nikitakarpei?tab=packages)

A lightweight, composable reimplementation of a [YaCy](https://yacy.net) peer, built as a set of small Go services. Each service runs independently, and a deployment wires together only what it needs. The services can also join the wider YaCy network over DHT.

The implementation follows its own design rather than mirroring the Java version: some features are still missing, while others take a different approach.

## Motivation

A standard YaCy node can exceed the memory available on small machines such as a Raspberry Pi, especially as the corpus grows. This project started with that constraint and became an experiment in a different way of building a YaCy peer.

## Build your own search stack

The core idea is runtime composability: independent pieces can be combined into different deployments.

[`doc/build-your-own-search-stack/`](doc/build-your-own-search-stack) explores this through a series of working deployments, starting with a single peer on a spare machine and gradually building toward a complete self-hosted search engine.

## Documentation

Each service under `services/` and plugin under `plugins/` has its own `doc/` directory:

* `specification.md` covers scope, requirements, and known limitations.
* `configuration.md` covers operator-facing settings.
