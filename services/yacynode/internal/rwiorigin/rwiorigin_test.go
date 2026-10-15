package rwiorigin_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

func TestOriginsFromOneNetworkShareIt(t *testing.T) {
	peer := yacymodel.WordHash("peer")

	for _, c := range []struct {
		name    string
		address string
		want    string
	}{
		{name: "ipv4 keeps 24 bits", address: "203.0.113.77", want: "203.0.113.0/24"},
		{name: "ipv6 keeps 56 bits", address: "2001:db8:aa:bbcc::1", want: "2001:db8:aa:bb00::/56"},
		{name: "ipv4 written as ipv6 keeps 24 bits", address: "::ffff:203.0.113.77", want: "203.0.113.0/24"},
		{name: "unparsable address stays as given", address: "unknown", want: "unknown"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := rwiorigin.Of(peer, c.address); got.Network != c.want || got.Peer != peer {
				t.Fatalf("Of(%q) = %+v, want network %q", c.address, got, c.want)
			}
		})
	}
}

var errBadRecord = errors.New("bad record")

func TestStoredOriginReadsBackAsWritten(t *testing.T) {
	written := rwiorigin.Of(yacymodel.WordHash("peer"), "203.0.113.77")
	var stored storedfields.Writer
	rwiorigin.Write(&stored, written)

	reader := storedfields.ReaderOf(stored.Record(), errBadRecord)
	if read := rwiorigin.From(reader); read != written || reader.Err() != nil {
		t.Fatalf("From = %+v (err %v), want %+v", read, reader.Err(), written)
	}
}

func TestRecordWithoutOriginReadsAsNoOrigin(t *testing.T) {
	reader := storedfields.ReaderOf([]byte{}, errBadRecord)

	if read := rwiorigin.From(reader); read != (rwiorigin.Origin{}) || reader.Err() != nil {
		t.Fatalf("From = %+v (err %v), want no origin", read, reader.Err())
	}
}

func TestOriginWithAMalformedPeerIsRejected(t *testing.T) {
	var stored storedfields.Writer
	stored.Bytes([]byte("short"))
	stored.Text("203.0.113.0/24")

	reader := storedfields.ReaderOf(stored.Record(), errBadRecord)
	rwiorigin.From(reader)
	if !errors.Is(reader.Err(), errBadRecord) {
		t.Fatalf("err = %v, want %v", reader.Err(), errBadRecord)
	}
}
