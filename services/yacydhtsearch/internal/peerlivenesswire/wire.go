// Package peerlivenesswire asks one peer address whether it is alive.
package peerlivenesswire

import (
	"context"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const livenessPath = "/yacy/query.html"

type Wire struct {
	client      *http.Client
	networkName string
}

func New(client *http.Client, networkName string) Wire {
	return Wire{client: client, networkName: networkName}
}

func (w Wire) Alive(ctx context.Context, address string) bool {
	probe := address + livenessPath + "?" + url.Values{
		yacyproto.FieldNetworkName: {w.networkName},
		yacyproto.FieldObject:      {string(yacyproto.ObjectRWICount)},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probe, nil)
	if err != nil {
		return false
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}
