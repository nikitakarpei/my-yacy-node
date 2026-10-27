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

Search the YaCy network at `http://127.0.0.1:8097`.

Crawl your own pages in the crawl console at `http://127.0.0.1:8093`. The search
page then also shows the pages that you crawled.

See the dashboards of the peer at `http://127.0.0.1:9090/dashboards`.

The search page, the crawl console and the dashboards link each other in a tab bar at the
top of each page.

To change these addresses, or to let other hosts reach them, see
[Configuration](doc/configuration.md).

## More

* [Configuration](doc/configuration.md): the settings of the peer and its
  services.
* [Specification](doc/specification.md): what the peer does, and its known
  limitations.
