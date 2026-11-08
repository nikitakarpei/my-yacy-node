package companionpageservices_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/brokerkey"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/pairingoffers"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/natsbroker"
)

const (
	secretLifetime = time.Minute
	pairingWait    = 30 * time.Second
	pageURL        = "http://origin.example/"
	pageWait       = 20 * time.Second
)

func TestAPairedDeviceBuildsTheRWIOfAPageWithThePageServicesPermissionsOnly(t *testing.T) {
	certificate, device := deviceCertificate(t)
	broker := brokerPairedWith(t, device)
	peer := inProcessJetStream(t, broker)
	provisionPeerStreams(t, peer)
	access := jetstreamconnect.BrokerAccess{
		TLS: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // the test pins nothing.
			Certificates:       []tls.Certificate{certificate},
		},
		InboxPrefix: companionprotocol.DeviceInboxPrefixOf(device),
	}
	brokerURL := "tls://" + broker.EndpointAddress().String()

	runPagescrape(t, brokerURL, access, pageServing(t))
	runPagerwi(t, brokerURL, access)

	pageRWIs, err := peer.CreateOrUpdateConsumer(t.Context(), pagerwicontract.PageRWIsStreamName,
		jetstream.ConsumerConfig{Durable: "node", AckPolicy: jetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatalf("open the RWI consumer: %v", err)
	}
	askForPage(t, peer)
	message, err := pageRWIs.Next(jetstream.FetchMaxWait(pageWait))
	if err != nil {
		t.Fatalf("await the RWI: %v", err)
	}
	pageRWI, err := pagerwicontract.UnmarshalPageRWI(message.Data())
	if err != nil || pageRWI.Metadata.Address != pageURL {
		t.Errorf("RWI = %+v (%v), want the RWI of %s", pageRWI.Metadata, err, pageURL)
	}
}

func deviceCertificate(t *testing.T) (tls.Certificate, companionprotocol.DeviceID) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
		companionprotocol.DeviceIDOf(leaf.RawSubjectPublicKeyInfo)
}

func brokerPairedWith(t *testing.T, device companionprotocol.DeviceID) *natsbroker.Broker {
	t.Helper()
	directory := t.TempDir()
	certificate, err := brokerkey.Settle(directory)
	if err != nil {
		t.Fatalf("Settle: %v", err)
	}
	devices, err := paireddevices.Open(directory)
	if err != nil {
		t.Fatalf("open the paired devices: %v", err)
	}
	t.Cleanup(func() { _ = devices.Close() })
	devices.Add(
		t.Context(),
		paireddevices.Device{ID: device, Name: "Pixel 8", PairedAt: time.Now()},
	)
	broker, err := natsbroker.StartWithEndpoint(t.TempDir(), natsbroker.Endpoint{
		Authentication: companionaccess.New(
			devices,
			pairingoffers.New(wallclock.Clock{}, secretLifetime, pairingWait),
			wallclock.Clock{},
			natspermissions.JoinedFrom(
				pagescrapeassembly.ServiceConfig{
					ScrapeRequestDurable: pagescrapeassembly.DefaultScrapeRequestDurable,
				}.BrokerPermissions(),
				pagerwiassembly.ServiceConfig{
					PageOfferDurable: pagerwiassembly.DefaultPageOfferDurable,
				}.BrokerPermissions(),
			),
			pairingWait,
		),
		Address:     "127.0.0.1:0",
		Certificate: certificate,
	})
	if err != nil {
		t.Fatalf("StartWithEndpoint: %v", err)
	}
	t.Cleanup(broker.Shutdown)
	return broker
}

func inProcessJetStream(t *testing.T, broker *natsbroker.Broker) jetstream.JetStream {
	t.Helper()
	js, connection, err := jetstreamconnect.Open(
		"nats://broker.internal",
		func(context.Context, string, string) (net.Conn, error) { return broker.Connection() },
	)
	if err != nil {
		t.Fatalf("connect in the process: %v", err)
	}
	t.Cleanup(connection.Close)
	return js
}

func provisionPeerStreams(t *testing.T, peer jetstream.JetStream) {
	t.Helper()
	for _, stream := range []jetstream.StreamConfig{
		{
			Name:      pagescrapecontract.ScrapeRequestsStreamName,
			Subjects:  []string{pagescrapecontract.ScrapeRequestSubject},
			Retention: jetstream.WorkQueuePolicy,
		},
		{
			Name: pagescrapecontract.ScrapePageOffersStreamName,
			Subjects: []string{
				pagescrapecontract.OfferedPageSubject,
				pagescrapecontract.ScrapeFailureSubject,
			},
			Retention: jetstream.InterestPolicy,
		},
		{
			Name:      pagerwicontract.PageRWIsStreamName,
			Subjects:  []string{pagerwicontract.PageRWISubject},
			Retention: jetstream.InterestPolicy,
		},
	} {
		if _, err := peer.CreateStream(t.Context(), stream); err != nil {
			t.Fatalf("create the %s stream: %v", stream.Name, err)
		}
	}
	for stream, consumer := range map[string]jetstream.ConsumerConfig{
		pagescrapecontract.ScrapeRequestsStreamName: {
			Durable:       pagescrapeassembly.DefaultScrapeRequestDurable,
			FilterSubject: pagescrapecontract.ScrapeRequestSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
		pagescrapecontract.ScrapePageOffersStreamName: {
			Durable:       pagerwiassembly.DefaultPageOfferDurable,
			FilterSubject: pagescrapecontract.OfferedPageSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
	} {
		if _, err := peer.CreateConsumer(t.Context(), stream, consumer); err != nil {
			t.Fatalf("create the %s consumer: %v", consumer.Durable, err)
		}
	}
}

func pageServing(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html lang="en"><body><p>alpha beta</p></body></html>`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func runService(t *testing.T, run func(ctx context.Context, health *servicehealth.Health) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	health := &servicehealth.Health{}
	done := make(chan error, 1)
	go func() { done <- run(ctx, health) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("service ended: %v", err)
		}
	})
}

func runPagescrape(
	t *testing.T,
	brokerURL string,
	access jetstreamconnect.BrokerAccess,
	proxyURL string,
) {
	t.Helper()
	environment := map[string]string{
		pagescrapeassembly.EnvScrapeNATSURL:    brokerURL,
		pagescrapeassembly.EnvProxyURL:         proxyURL,
		pagescrapeassembly.EnvProxyDialMode:    pagefetchershttp.ProxyDialAbsoluteURLName,
		pagescrapeassembly.EnvProvisionStreams: strconv.FormatBool(false),
	}
	cfg, err := pagescrapeassembly.LoadServiceConfig(func(name string) string {
		return environment[name]
	})
	if err != nil {
		t.Fatalf("pagescrape config: %v", err)
	}
	cfg.BrokerAccess = access
	runService(t, func(ctx context.Context, health *servicehealth.Health) error {
		return pagescrapeassembly.RunService(ctx, cfg, pagescrapeassembly.Listeners{},
			(&net.Dialer{}).DialContext, prometheus.NewRegistry(), health)
	})
}

func runPagerwi(t *testing.T, brokerURL string, access jetstreamconnect.BrokerAccess) {
	t.Helper()
	environment := map[string]string{
		pagerwiassembly.EnvPageOfferNATSURL: brokerURL,
		pagerwiassembly.EnvPageRWINATSURL:   brokerURL,
		pagerwiassembly.EnvProvisionStreams: strconv.FormatBool(false),
	}
	cfg, err := pagerwiassembly.LoadServiceConfig(func(name string) string {
		return environment[name]
	})
	if err != nil {
		t.Fatalf("pagerwi config: %v", err)
	}
	cfg.BrokerAccess = access
	runService(t, func(ctx context.Context, health *servicehealth.Health) error {
		return pagerwiassembly.RunService(ctx, cfg, pagerwiassembly.Listeners{},
			(&net.Dialer{}).DialContext, prometheus.NewRegistry(), health)
	})
}

func askForPage(t *testing.T, peer jetstream.JetStream) {
	t.Helper()
	data, err := pagescrapecontract.MarshalScrapeRequest(pagescrapecontract.ScrapeRequest{
		PageURL: canonicalurltest.CanonicalURLOf(t, pageURL),
	})
	if err != nil {
		t.Fatalf("marshal the scrape request: %v", err)
	}
	if _, err := peer.Publish(
		t.Context(),
		pagescrapecontract.ScrapeRequestSubject,
		data,
	); err != nil {
		t.Fatalf("publish the scrape request: %v", err)
	}
}
