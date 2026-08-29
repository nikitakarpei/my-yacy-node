package jetstreamconnect_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
)

func TestOpenConnectsAndCloses(t *testing.T) {
	url := natstestserver.Start(t)
	js, connection, err := jetstreamconnect.Open(url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if js == nil {
		t.Fatal("Open returned nil jetstream")
	}
	connection.Close()
}

func TestOpenRejectsUnreachable(t *testing.T) {
	if _, _, err := jetstreamconnect.Open("nats://127.0.0.1:1"); err == nil {
		t.Fatal("expected error for unreachable server")
	}
}
