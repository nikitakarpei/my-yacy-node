package jetstreamconnect

import (
	"crypto/tls"

	"github.com/nats-io/nats.go"
)

type BrokerAccess struct {
	TLS         *tls.Config
	InboxPrefix string
}

func (a BrokerAccess) Options() []nats.Option {
	var options []nats.Option
	if a.TLS != nil {
		options = append(options, nats.Secure(a.TLS))
	}
	if a.InboxPrefix != "" {
		options = append(options, nats.CustomInboxPrefix(a.InboxPrefix))
	}
	return options
}
