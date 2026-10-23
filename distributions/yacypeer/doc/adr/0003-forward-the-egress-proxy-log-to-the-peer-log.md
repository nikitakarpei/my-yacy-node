# 3. Forward the egress proxy log to the peer log

Status: accepted

## Context

The built-in egress proxy writes its log through a `logrus` logger. The services of the peer
write their log through `log/slog`, at the level that `LOG_LEVEL` sets.

`github.com/sirupsen/logrus` is under the MIT licence. Smokescreen already requires it.

## Decision

Give the egress proxy a `logrus` logger that forwards each entry to the peer log.

## Consequences

* The peer writes one log format.
* `LOG_LEVEL` controls the egress proxy log too.
* A refused request is a warning, and other proxy entries are debug entries.
