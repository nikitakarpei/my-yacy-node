package proxyintake

import "context"

type ProxyResponseDeliveryObserver interface {
	ProxyResponseDelivered(ctx context.Context, targetURL string)
	ProxyResponseDeliveryFailed(ctx context.Context, targetURL string, cause error)
}

type ProxyResponseDeliveryObservers []ProxyResponseDeliveryObserver

func (observers ProxyResponseDeliveryObservers) ProxyResponseDelivered(
	ctx context.Context,
	targetURL string,
) {
	for _, observer := range observers {
		observer.ProxyResponseDelivered(ctx, targetURL)
	}
}

func (observers ProxyResponseDeliveryObservers) ProxyResponseDeliveryFailed(
	ctx context.Context,
	targetURL string,
	cause error,
) {
	for _, observer := range observers {
		observer.ProxyResponseDeliveryFailed(ctx, targetURL, cause)
	}
}
