# YaCy Companion — Configuration

The app has no settings. It takes all that it needs from the pairing code, a QR code that holds
this link:

```text
yacypeer-companion://connect?nats=<NATS URL>&partition-exponent=<exponent>
```

| Parameter | Meaning |
|---|---|
| `nats` | NATS URL of the broker of the peer, URL-encoded, with the scheme `nats` or `tls`. |
| `partition-exponent` | Partition exponent of the YaCy network of the peer, from 0 to 62. |

The app sets the other environment variables of pagescrape and pagerwi to their defaults.
pagescrape fetches through the capture proxy of the app.
