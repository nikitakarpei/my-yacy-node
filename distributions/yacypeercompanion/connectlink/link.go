// Package connectlink reads the link that yacypeer shows as a QR code. The link names the
// broker of the peer and the partition exponent of its YaCy network.
package connectlink

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	Scheme                     = "yacypeer-companion"
	Host                       = "connect"
	ParameterNATSURL           = "nats"
	ParameterPartitionExponent = "partition-exponent"
)

var errNotAConnectLink = errors.New("not a yacypeer connect link")

type Link struct {
	NATSURL           string
	PartitionExponent int
}

func LinkFrom(text string) (*Link, error) {
	link, err := url.Parse(text)
	if err != nil || link.Scheme != Scheme || link.Host != Host {
		return nil, errNotAConnectLink
	}
	parameters := link.Query()
	natsURL, err := natsURLFrom(parameters.Get(ParameterNATSURL))
	if err != nil {
		return nil, err
	}
	partitionExponent, err := partitionExponentFrom(parameters.Get(ParameterPartitionExponent))
	if err != nil {
		return nil, err
	}
	return &Link{NATSURL: natsURL, PartitionExponent: partitionExponent}, nil
}

func natsURLFrom(text string) (string, error) {
	natsURL, err := url.Parse(text)
	if err != nil || (natsURL.Scheme != "nats" && natsURL.Scheme != "tls") || natsURL.Host == "" {
		return "", fmt.Errorf("%w: %s is not a NATS URL", errNotAConnectLink, ParameterNATSURL)
	}
	return text, nil
}

func partitionExponentFrom(text string) (int, error) {
	partitionExponent, err := strconv.ParseUint(text, 10, 8)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s is not a number",
			errNotAConnectLink,
			ParameterPartitionExponent,
		)
	}
	if _, err := yacymodel.DHTRingPartitionsFromExponent(uint(partitionExponent)); err != nil {
		return 0, fmt.Errorf("%w: %s: %w", errNotAConnectLink, ParameterPartitionExponent, err)
	}
	return int(partitionExponent), nil
}
