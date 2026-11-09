package pageservices_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	pagerwiassembly "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/brokeraccess/brokeraccesstest"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/pageservices"
)

const (
	runWait    = 10 * time.Second
	pageURL    = "http://origin.example/"
	appVersion = "1.2.0"
	usedSecret = "used-secret"

	peerPartitionExponent = 6
)

type observedEvents chan string

func (events observedEvents) ServicesConnected()    { events <- "connected" }
func (events observedEvents) BrokerConnectionLost() { events <- "connection lost" }
func (events observedEvents) ServicesStopped()      { events <- "stopped" }
func (events observedEvents) BrokerUnreachable()    { events <- "broker unreachable" }
func (events observedEvents) ServicesFailed()       { events <- "failed" }
func (events observedEvents) DeviceRefused()        { events <- "refused" }
func (events observedEvents) AppOutdated()          { events <- "app outdated" }
func (events observedEvents) PeerOutdated()         { events <- "peer outdated" }

type deviceCheck struct {
	refused                 atomic.Bool
	refusesOnceItHasSettled atomic.Bool
}

func (c *deviceCheck) Check(client natsserver.ClientAuthentication) bool {
	if client.GetOpts().Username == brokeraccesstest.PeerUser {
		return true
	}
	return !c.refused.Load() && len(client.GetTLSConnectionState().PeerCertificates) > 0
}

type peer struct {
	broker      brokeraccesstest.Broker
	check       *deviceCheck
	appVersions chan string
}

func startPeer(t *testing.T) peer {
	t.Helper()
	return startPeerReplyingToHello(t, companionprotocol.AcceptedHelloReply())
}

func startPeerReplyingToHello(t *testing.T, helloReply companionprotocol.HelloReply) peer {
	t.Helper()
	check := &deviceCheck{}
	broker := brokeraccesstest.Start(t, check)
	appVersions := make(chan string, 4)
	connection := broker.Connect(t)
	settings, _ := json.Marshal(
		companionprotocol.DeviceSettings{PartitionExponent: peerPartitionExponent},
	)
	reply, _ := json.Marshal(helloReply)
	for subject, handler := range map[string]nats.MsgHandler{
		companionprotocol.HelloSubjects: func(msg *nats.Msg) {
			var hello companionprotocol.Hello
			_ = json.Unmarshal(msg.Data, &hello)
			appVersions <- hello.AppVersion
			_ = msg.Respond(reply)
		},
		companionprotocol.SettingsSubjects: func(msg *nats.Msg) {
			if check.refusesOnceItHasSettled.Load() {
				check.refused.Store(true)
			}
			_ = msg.Respond(settings)
		},
	} {
		if _, err := connection.Subscribe(subject, handler); err != nil {
			t.Fatalf("Subscribe: %v", err)
		}
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	return peer{broker: broker, check: check, appVersions: appVersions}
}

func (p peer) jetStream(t *testing.T) jetstream.JetStream {
	t.Helper()
	js, err := jetstream.New(p.broker.Connect(t))
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return js
}

func (p peer) kickEveryDevice(t *testing.T) {
	t.Helper()
	p.check.refused.Store(true)
	connections, err := p.broker.Server.Connz(&natsserver.ConnzOptions{Username: true})
	if err != nil {
		t.Fatalf("Connz: %v", err)
	}
	for _, connection := range connections.Conns {
		if connection.AuthorizedUser != brokeraccesstest.PeerUser {
			_ = p.broker.Server.DisconnectClientByID(connection.Cid)
		}
	}
}

func instanceFor(t *testing.T, linkText string, captureProxyURL string) *pageservices.Instance {
	t.Helper()
	instance, err := pageservices.InstanceFor(
		linkText, captureProxyURL, brokeraccesstest.RandomDeviceKey(t), appVersion,
	)
	if err != nil {
		t.Fatalf("InstanceFor: %v", err)
	}
	return instance
}

func captureProxyServingThePage(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(pageBody))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func awaitEvent(t *testing.T, events observedEvents, want string) {
	t.Helper()
	select {
	case event := <-events:
		if event != want {
			t.Fatalf("event = %q, want %q", event, want)
		}
	case <-time.After(runWait):
		t.Fatalf("no %q event", want)
	}
}

const pageBody = "<html lang=\"en\"><body><p>alpha beta</p></body></html>"

func TestInstanceForRefusesTextThatIsNotAPairLink(t *testing.T) {
	if _, err := pageservices.InstanceFor(
		"https://example.org/",
		"http://127.0.0.1:1",
		brokeraccesstest.RandomDeviceKey(t),
		appVersion,
	); err == nil {
		t.Fatal("want an error for text that is not a pair link")
	}
}

func TestInstanceBuildsTheRWIOfAPageThatTheBrokerAsksFor(t *testing.T) {
	peer := startPeer(t)
	provisionPeerStreams(t, peer.jetStream(t))
	instance := instanceFor(t, peer.broker.LinkWith(usedSecret), captureProxyServingThePage(t))
	events := make(observedEvents, 4)
	go instance.Run(events)
	awaitEvent(t, events, "connected")

	pageRWI := askForRWIOfPage(t, peer.jetStream(t))
	if pageRWI.Metadata.Address != pageURL {
		t.Errorf("RWI address = %q, want %q", pageRWI.Metadata.Address, pageURL)
	}
	if pageRWI.Metadata.PostingsRoot != postingsRootIn(t, peerPartitionExponent, pageRWI) {
		t.Error("the RWI is built for another partition exponent than the one of the peer")
	}
	if reported := <-peer.appVersions; reported != appVersion {
		t.Errorf("reported app version = %q, want %q", reported, appVersion)
	}

	instance.Stop()
	awaitEvent(t, events, "stopped")
}

func postingsRootIn(
	t *testing.T,
	partitionExponent uint,
	pageRWI pagerwicontract.PageRWI,
) yacymodel.Optional[yacymodel.PostingsRoot] {
	t.Helper()
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("partitions: %v", err)
	}
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(
		pageRWI.Postings, yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		t.Fatalf("postings of the RWI: %v", err)
	}
	return pagePostings.Root()
}

func TestInstanceRunReportsADeviceThatThePeerRefuses(t *testing.T) {
	peer := startPeer(t)
	peer.check.refused.Store(true)
	instance := instanceFor(t, peer.broker.LinkWith(usedSecret), "http://127.0.0.1:1")
	events := make(observedEvents, 4)

	instance.Run(events)

	awaitEvent(t, events, "refused")
}

func TestInstanceRunReportsADeviceThatThePeerRefusesOnceItHasSettled(t *testing.T) {
	peer := startPeer(t)
	provisionPeerStreams(t, peer.jetStream(t))
	peer.check.refusesOnceItHasSettled.Store(true)
	instance := instanceFor(t, peer.broker.LinkWith(usedSecret), captureProxyServingThePage(t))
	events := make(observedEvents, 4)

	instance.Run(events)

	awaitEvent(t, events, "refused")
}

func TestInstanceRunReportsADeviceThatThePeerRefusesWhileItWorks(t *testing.T) {
	peer := startPeer(t)
	provisionPeerStreams(t, peer.jetStream(t))
	instance := instanceFor(t, peer.broker.LinkWith(usedSecret), captureProxyServingThePage(t))
	events := make(observedEvents, 8)
	go instance.Run(events)
	awaitEvent(t, events, "connected")

	peer.kickEveryDevice(t)

	for {
		select {
		case event := <-events:
			switch event {
			case "connection lost":
				continue
			case "refused":
				return
			default:
				t.Fatalf("event = %q, want the device reported refused", event)
			}
		case <-time.After(runWait):
			t.Fatal("the run did not end after the peer refused the device")
		}
	}
}

func provisionPeerStreams(t *testing.T, broker jetstream.JetStream) {
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
		if _, err := broker.CreateStream(t.Context(), stream); err != nil {
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
		if _, err := broker.CreateConsumer(t.Context(), stream, consumer); err != nil {
			t.Fatalf("create the %s consumer: %v", consumer.Durable, err)
		}
	}
}

func askForRWIOfPage(t *testing.T, broker jetstream.JetStream) pagerwicontract.PageRWI {
	t.Helper()
	pageRWIs, err := broker.CreateOrUpdateConsumer(t.Context(), pagerwicontract.PageRWIsStreamName,
		jetstream.ConsumerConfig{Durable: "node", AckPolicy: jetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatalf("open the RWI consumer: %v", err)
	}
	askForPage(t, broker)
	message, err := pageRWIs.Next(jetstream.FetchMaxWait(runWait))
	if err != nil {
		t.Fatalf("await the RWI: %v", err)
	}
	pageRWI, err := pagerwicontract.UnmarshalPageRWI(message.Data())
	if err != nil {
		t.Fatalf("unmarshal the RWI: %v", err)
	}
	return pageRWI
}

func TestInstanceRunReportsStreamsThatThePeerHasNotCreated(t *testing.T) {
	peer := startPeer(t)
	instance := instanceFor(t, peer.broker.LinkWith(usedSecret), "http://127.0.0.1:1")
	events := make(observedEvents, 4)
	instance.Run(events)
	awaitEvent(t, events, "failed")
}

func TestInstanceRunReportsAnAppOrPeerThatIsOutdatedForTheOther(t *testing.T) {
	for reason, want := range map[companionprotocol.RefusalReason]string{
		companionprotocol.ReasonAppOutdated:    "app outdated",
		companionprotocol.ReasonPeerOutdated:   "peer outdated",
		companionprotocol.ReasonRequestInvalid: "failed",
	} {
		t.Run(string(reason), func(t *testing.T) {
			peer := startPeerReplyingToHello(t, companionprotocol.RefusedHelloReply(reason))
			instance := instanceFor(t, peer.broker.LinkWith(usedSecret), "http://127.0.0.1:1")
			events := make(observedEvents, 4)

			instance.Run(events)

			awaitEvent(t, events, want)
		})
	}
}

func TestInstanceRunReportsAPeerThatAnswersNothingAsFailed(t *testing.T) {
	broker := brokeraccesstest.Start(t, &deviceCheck{})
	instance := instanceFor(t, broker.LinkWith(usedSecret), "http://127.0.0.1:1")
	events := make(observedEvents, 4)
	instance.Run(events)
	awaitEvent(t, events, "failed")
}

func TestInstanceRunReportsAnUnreachableBroker(t *testing.T) {
	link := companionprotocol.PairLink{
		Addresses: []string{"127.0.0.1:1"}, BrokerKeyHash: "hash", Secret: usedSecret,
	}
	instance := instanceFor(t, link.String(), "http://127.0.0.1:1")
	events := make(observedEvents, 4)
	instance.Run(events)
	awaitEvent(t, events, "broker unreachable")
}

func TestInstanceRunReportsABrokerThatShutsDown(t *testing.T) {
	peer := startPeer(t)
	provisionPeerStreams(t, peer.jetStream(t))
	instance := instanceFor(t, peer.broker.LinkWith(usedSecret), captureProxyServingThePage(t))
	events := make(observedEvents, 4)
	go instance.Run(events)
	awaitEvent(t, events, "connected")
	askForRWIOfPage(t, peer.jetStream(t))

	peer.broker.Server.Shutdown()
	for {
		select {
		case event := <-events:
			switch event {
			case "connection lost":
				continue
			case "broker unreachable":
				return
			default:
				t.Fatalf("event = %q, want the broker reported unreachable", event)
			}
		case <-time.After(runWait):
			t.Fatal("the run did not end after the broker shut down")
		}
	}
}

func TestInstanceRunReportsAStopBeforeItStarted(t *testing.T) {
	peer := startPeer(t)
	instance := instanceFor(t, peer.broker.LinkWith(usedSecret), "http://127.0.0.1:1")
	instance.Stop()
	events := make(observedEvents, 4)
	instance.Run(events)
	awaitEvent(t, events, "stopped")
}

func askForPage(t *testing.T, broker jetstream.JetStream) {
	t.Helper()
	data, err := pagescrapecontract.MarshalScrapeRequest(pagescrapecontract.ScrapeRequest{
		PageURL: canonicalurltest.CanonicalURLOf(t, pageURL),
	})
	if err != nil {
		t.Fatalf("marshal the scrape request: %v", err)
	}
	if _, err := broker.Publish(
		t.Context(),
		pagescrapecontract.ScrapeRequestSubject,
		data,
	); err != nil {
		t.Fatalf("publish the scrape request: %v", err)
	}
}
