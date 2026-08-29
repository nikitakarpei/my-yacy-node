# 12. Watch the stack on dashboards

> "Can I see all of it at once instead?"

Grafana, with the Prometheus datasource and two dashboards already provisioned:
one for the peer, one for the crawler. They are the same JSON files that live
beside those services in `services/*/doc/`, mounted read-only, so a dashboard
shipped with a service is the dashboard you see.

Anonymous access is on and the login form is off. That is fine on `127.0.0.1`,
which is where it is published; it is not fine on an address anyone else can
reach. If you put Grafana behind Caddy, put authentication in front of it first.

## Try it

```sh
docker compose up -d
```

Open `http://localhost:3000`. No login.

## Where you are now

You have a search engine that crawls what you read, an index you can query, a
peer trading postings with the rest of the network, an AI assistant working
through all of it, and enough instruments to see it working.

The rest is tuning. `side-roads/` holds the alternatives worth knowing about,
and every `configuration.md` under `services/` documents settings this journey
left at their defaults.
