# YaCy peer only

Runs one YaCy RWI node as a peer on the YaCy network, joining the DHT and answering remote
search requests. This is the smallest deployment: no crawler and no search UI, so it
contributes storage and search capacity to the network and nothing else.

## Services

| Service | Role |
| --- | --- |
| `yacy-rwi-node` | The peer: joins the DHT and serves remote search requests. |
| `smokescreen` | Egress proxy every outbound connection passes through; blocks private and internal IP addresses. |

## Setup

1. Copy `.env.example` to `.env` and set `YACY_PEER_HASH`, `YACY_PEER_NAME`, and
   `YACY_ADVERTISE_HOST`.
2. Copy `docker-compose.yml.example` to `docker-compose.yml`.
3. Start the stack: `docker compose up -d`.
