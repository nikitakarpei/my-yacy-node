package jetstreamconnect

import (
	"crypto/tls"

	"github.com/nats-io/nats.go"
)

type BrokerAccess struct {
	TLS         *tls.Config
	ClientName  string
	Password    string
	InboxPrefix string
}

func (a BrokerAccess) Options() []nats.Option {
	var options []nats.Option
	if a.TLS != nil {
		options = append(options, nats.Secure(a.TLS))
	}
	if a.ClientName != "" {
		options = append(options, nats.Name(a.ClientName))
	}
	if a.Password != "" {
		options = append(options, nats.UserInfo("", a.Password))
	}
	if a.InboxPrefix != "" {
		options = append(options, nats.CustomInboxPrefix(a.InboxPrefix))
	}
	return options
}
