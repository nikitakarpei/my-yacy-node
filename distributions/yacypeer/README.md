# yacypeer

yacypeer runs a YaCy peer as one program. The peer joins the YaCy network,
searches it, crawls your own pages, and searches the pages it crawled. Use it to
try the peer before you compose its services yourself.

## Build

Install Go. Then, from the repository root, run:

```sh
cd distributions/yacypeer
go build ./cmd/yacypeer
```

## Run

```sh
./yacypeer
```

With no settings, the peer joins the YaCy network. Other peers must reach port
8090 of the host from the internet. Behind NAT, forward that port to the host.

## Use

Search the YaCy network on the search page at `http://127.0.0.1:8097`. When you open
a result, the peer crawls that page and adds it to its index.

Crawl your own pages in the crawl console at `http://127.0.0.1:8093`. Then
search the pages that you crawled:

```sh
curl 'http://127.0.0.1:8096/search?query=yacy'
```

To change these addresses, or to let other hosts reach them, see
[Configuration](doc/configuration.md).

## More

* [Configuration](doc/configuration.md): the settings of the peer and its
  services.
* [Specification](doc/specification.md): what the peer does, and its known
  limitations.
