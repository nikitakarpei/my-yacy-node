# yacypeer

yacypeer runs a YaCy peer as one program. The peer joins the YaCy network,
searches it, crawls your own pages, and searches the pages it crawled. Use it to
try the peer before you compose its services yourself.

## Run

Download the binary for your system from the
[latest release](https://github.com/nikitakarpei/my-yacy-node/releases/latest).
On Linux and macOS, make it executable. Then start it in the directory that keeps
the peer data:

```sh
chmod +x yacypeer-linux-amd64
./yacypeer-linux-amd64
```

Other peers must reach port 8090 of the host from the internet. Behind NAT,
forward this port to the host. The port carries HTTP and TLS between peers.

The first start searches the peer key for one minute. The key is in `peer.key`
of the `data` directory. Keep a copy of this file: the peer hash and the data of
the peer belong to the key.

## Use

Go to `http://127.0.0.1:9090` in your browser. The tabs at the top take you to the search page, the crawl console, and the dashboards. Searches cover the whole YaCy network plus any websites you’ve crawled yourself in the crawl console.

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
