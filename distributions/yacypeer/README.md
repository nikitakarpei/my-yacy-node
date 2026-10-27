# yacypeer

yacypeer runs a YaCy peer as one program. The peer joins the YaCy network,
searches it, crawls your own pages, and searches the pages it crawled. Use it to
try the peer before you compose its services yourself.

## Run

Install Docker or Podman. Then run the command below. With Podman, write `podman`
for `docker`.

```sh
docker run -d --name yacypeer --restart unless-stopped \
  -p 8090:8090 -p 127.0.0.1:9090:9090 \
  -v yacypeer:/var/lib/yacypeer \
  ghcr.io/nikitakarpei/my-yacy-node/yacypeer:latest
```

With no settings, the peer joins the YaCy network. Other peers must reach port
8090 of the host from the internet. Behind NAT, forward that port to the host.
The `yacypeer` volume keeps the data of the peer.

## Use

Open `http://127.0.0.1:9090`. The tabs at the top link the search page, the crawl
console and the dashboards. The search page searches the YaCy network and the pages
that you crawl in the crawl console.

To change this address, or to let other hosts reach it, see
[Configuration](doc/configuration.md).

## Build from source

Install Go. Then, from the repository root, run:

```sh
cd distributions/yacypeer
go build ./cmd/yacypeer
./yacypeer
```

## More

* [Configuration](doc/configuration.md): the settings of the peer and its
  services.
* [Specification](doc/specification.md): what the peer does, and its known
  limitations.
