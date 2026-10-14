# Peer messages

The operator of a YaCy peer can send a message to the operator of a different
peer. The node gives each message that it accepts to a webhook that you set.
The node does not keep messages and does not send a message again.

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_MESSAGE_WEBHOOK_URL` | _(empty)_ | The `http` or `https` URL that gets each accepted message. Leave it empty to refuse all messages. |

## Accepted messages

The node accepts a message only when all these conditions are true:

- The seed of the sender has the peer hash that the sender says.
- The subject and the text are not empty.
- The subject has 100 characters or fewer.
- The text has 10240 characters or fewer.
- The address of the sender has messages left for this hour.
- The webhook accepts the message.

Each sender address can send 5 messages in one hour. The hour starts with the
first message from that address. The node counts at most 1024 sender addresses.
When it counts 1024 addresses, it refuses a new address until the hour of an
address ends.

The sender sees `Thank you!` when the webhook accepts the message. Otherwise,
the sender sees `-1`.

## Webhook request

The node sends one `POST` request with this JSON body:

```json
{
  "subject": "your peer",
  "body": "sends me spam",
  "sender": {
    "hash": "WSjicAx1hRio",
    "name": "zeus",
    "address": "203.0.113.7"
  }
}
```

The `address` is the IP address that sent the message. The sender gives the
`hash` and the `name`. The node cannot make sure that they are correct.

A response with a status from 200 to 299 accepts the message. The node does not
follow redirects. The node stops a delivery after 10 seconds.

## Route to the webhook

Before each message, the node finds the IP addresses of the webhook host.

When all the addresses are private, the node connects directly to these
addresses. The URL can use `http`. Private addresses are loopback, link-local,
RFC 1918, IPv6 unique local, and `100.64.0.0/10` addresses.

When one or more addresses are public, the node connects through the egress
proxy in `EGRESS_PROXY_URL`. The URL must use `https`. If the URL uses `http`,
the node does not send the message and logs an error.
