package pageservices_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/predicates/httpresponse"

	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck/pageattestationchecktest"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/pageservices"
)

const (
	attestedPageURL = "https://origin.example/"
	quicknetRound   = 33667957
	roundSignature  = "ace44ad88121ed26163c00acaa4c89522295c4fb066a8e9712138c63644e43cadac17e3f29aa8056f8f4ef5449d6e16e"
)

var errNoHardwareKey = errors.New("no hardware key in this test")

func TestInstancePublishesAnRWIWhosePageAttestationAPeerAccepts(t *testing.T) {
	device := pageattestationchecktest.NewAttestingDevice(
		t,
		pageattestationchecktest.AppVersionCode,
	)
	authority := pageattestationchecktest.NewTimeStampAuthority(t)
	peer := startPeer(t)
	provisionPeerStreams(t, peer.jetStream(t))
	instance := instanceWithKeys(
		t, peer.broker.LinkWith(usedSecret), captureProxySigningWith(t, device),
		deviceKeys{t: t, device: device, authority: authority},
		deviceKeys{t: t, device: device, authority: authority},
	)
	events := make(observedEvents, 4)
	go instance.Run(events)
	awaitEvent(t, events, "connected")

	pageRWI := askForRWIOf(t, peer.jetStream(t), attestedPageURL)
	verdict := checkFor(t, device, authority).VerdictOn(pageRWI.Metadata, pageRWI.Attestation)

	if !verdict.Attested {
		t.Fatalf("verdict = %+v, want the page attestation accepted", verdict)
	}
	if want := quicknet(t).RoundTime(quicknetRound); !verdict.FetchedAt.Equal(want) {
		t.Errorf("fetched at = %v, want the time of the capture round %v", verdict.FetchedAt, want)
	}

	instance.Stop()
	awaitEvent(t, events, "stopped")
}

func checkFor(
	t *testing.T,
	device *pageattestationchecktest.AttestingDevice,
	authority *pageattestationchecktest.TimeStampAuthority,
) pageattestationcheck.Check {
	t.Helper()
	now := time.Now()
	release := trustedappreleases.TrustedRelease{
		Tag:                "yacypeercompanion-v1.0.0",
		SigningCertificate: pageattestationchecktest.AppSigningCertificate,
		VersionCode:        pageattestationchecktest.AppVersionCode,
	}

	return pageattestationcheck.New(
		authority.Entry(), []trustedappreleases.TrustedRelease{release}, now, device.Trust(), now,
	)
}

func captureProxySigningWith(
	t *testing.T,
	device *pageattestationchecktest.AttestingDevice,
) string {
	t.Helper()
	statement := responseStatementOf(t, attestedPageURL, pageBody)
	signature, err := device.Sign(verifier.PAE(verifier.InTotoPayload, statement))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Attested-Version", "v0.1")
		w.Header().Set("Attested-Statement", byteSequenceOf(statement))
		w.Header().Set("Attested-Signature", byteSequenceOf(signature))
		w.Header().Set("Attested-Key", byteSequenceOf(device.KeyEvidence()))
		_, _ = w.Write([]byte(pageBody))
	}))
	t.Cleanup(server.Close)

	return server.URL
}

func responseStatementOf(t *testing.T, requestURL, body string) []byte {
	t.Helper()
	bodyDigest := sha256.Sum256([]byte(body))
	contentType := "text/html"
	statement, err := json.Marshal(map[string]any{
		"_type": verifier.InTotoStatement1,
		"subject": []map[string]any{{
			"name":      "body",
			"digest":    map[string]string{"sha256": hex.EncodeToString(bodyDigest[:])},
			"mediaType": contentType,
		}},
		"predicateType": httpresponse.PredicateType,
		"predicate": map[string]any{
			"request":  map[string]any{"method": "GET", "url": requestURL, "cookies": "none"},
			"response": map[string]any{"statusLine": "HTTP/1.1 200 OK", "status": http.StatusOK},
			"startedAfter": map[string]any{"drand": map[string]any{
				"chain": quicknet(t).Hash, "round": quicknetRound, "signature": roundSignature,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	return statement
}

func byteSequenceOf(content []byte) string {
	return ":" + base64.StdEncoding.EncodeToString(content) + ":"
}

func quicknet(t *testing.T) verifier.DrandChain {
	t.Helper()
	chain, err := verifier.DrandQuicknet()
	if err != nil {
		t.Fatal(err)
	}

	return chain
}

type deviceKeys struct {
	t         *testing.T
	device    *pageattestationchecktest.AttestingDevice
	authority *pageattestationchecktest.TimeStampAuthority
}

func (k deviceKeys) CurrentKey() (pageservices.HardwareKey, error) {
	return k.device, nil
}

func (k deviceKeys) Stamp(signedStatementID []byte) ([]byte, error) {
	return k.authority.TimeStampEncodingOf(k.t, signedStatementID, time.Now()), nil
}

type unavailableHardwareKeys struct{}

func (unavailableHardwareKeys) CurrentKey() (pageservices.HardwareKey, error) {
	return nil, errNoHardwareKey
}

func (unavailableHardwareKeys) Stamp([]byte) ([]byte, error) {
	return nil, errNoHardwareKey
}
