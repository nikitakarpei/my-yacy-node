# YaCy Companion — Configuration

The app has no settings. It takes all that it needs from the pairing code, a QR code that holds
this link:

```text
yacypeer-companion://pair?name=<name>&address=<address>&brokerKeyHash=<hash>&secret=<secret>
```

| Parameter | Meaning |
|---|---|
| `name` | mDNS name of the host of the peer. The app uses it when `address` does not answer. |
| `address` | Address and port of the broker of the peer. |
| `brokerKeyHash` | SHA-256 hash of the TLS public key of the broker, base64url. |
| `secret` | One-time secret of the pairing. It expires when the pairing page shows a new code. |

After pairing, the peer gives the partition exponent of its YaCy network each time the app
connects. The app sets the other environment variables of pagescrape and pagerwi to their
defaults. pagescrape fetches through the capture proxy of the app.
