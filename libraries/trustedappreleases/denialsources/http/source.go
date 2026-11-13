// Package http reads release denials from a JSON document at an address, such as a file of a
// repository on its default branch.
package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const maxDenialsBytes = 1 << 20

var (
	ErrUnexpectedStatus = errors.New(
		"the server of the release denials answered with an unexpected status",
	)
	ErrTooLarge = errors.New("the release denials are too large")
)

type Source struct {
	client  *http.Client
	address string
}

func New(client *http.Client, address string) Source {
	return Source{client: client, address: address}
}

func (s Source) Denials(ctx context.Context) ([]trustedappreleases.Denial, error) {
	content, err := s.content(ctx)
	if err != nil {
		return nil, err
	}

	return trustedappreleases.DenialsFrom(content)
}

func (s Source) content(ctx context.Context) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.address, nil)
	if err != nil {
		return nil, fmt.Errorf("ask for the release denials at %s: %w", s.address, err)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("ask for the release denials at %s: %w", s.address, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d at %s", ErrUnexpectedStatus, response.StatusCode, s.address)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxDenialsBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read the release denials at %s: %w", s.address, err)
	}
	if len(content) > maxDenialsBytes {
		return nil, fmt.Errorf("%w: %s", ErrTooLarge, s.address)
	}

	return content, nil
}
