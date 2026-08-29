# What this is

[![ci](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/nikitakarpei/my-yacy-node/actions/workflows/ci.yml)
[![go](https://img.shields.io/github/go-mod/go-version/nikitakarpei/my-yacy-node?filename=services%2Fyacynode%2Fgo.mod)](https://github.com/nikitakarpei/my-yacy-node/blob/main/services/yacynode/go.mod)
[![license](https://img.shields.io/github/license/nikitakarpei/my-yacy-node)](LICENSE)
[![images](https://img.shields.io/badge/ghcr.io-amd64%20%7C%20arm64-blue)](https://github.com/nikitakarpei?tab=packages)

A lightweight reimplementation of a [YaCy](https://yacy.net) peer, built as
small, single-purpose Go services.

> “YaCy is a distributed Web Search Engine, based on a peer-to-peer network.”
> — [YaCy FAQ](https://yacy.net/faq/)

Start with [Build your own search stack](doc/build-your-own-search-stack) to see
how to compose the services into working deployments.

## Motivation

A standard YaCy node can exceed the memory available on small machines such as
a Raspberry Pi, especially as the corpus grows. This project started with that
constraint and became an experiment in a different way of building a YaCy peer.

## Documentation

Each service under `services/` and plugin under `plugins/` has its own `doc/`
directory:

- `specification.md` covers scope, requirements, and known limitations.
- `configuration.md` covers operator-facing settings.

## Project status

This project is under active development. It follows its own design instead of
mirroring the Java version. Some features are not available yet, and others
work differently.
