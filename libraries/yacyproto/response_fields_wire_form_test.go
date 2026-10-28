package yacyproto_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestParseResponseFields(t *testing.T) {
	fields := yacyproto.ParseResponseFields([]byte("version=1.0\r\nuptime=42\n\nresult=ok\n"))
	if fields["version"] != "1.0" {
		t.Errorf("version = %q", fields["version"])
	}
	if fields["uptime"] != "42" {
		t.Errorf("uptime = %q", fields["uptime"])
	}
	if fields["result"] != "ok" {
		t.Errorf("result = %q", fields["result"])
	}
	if len(fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(fields))
	}
}

func TestParseResponseFieldsIgnoresMalformedLines(t *testing.T) {
	fields := yacyproto.ParseResponseFields([]byte("noeq\n=novalue\nok=yes\n"))
	if len(fields) != 1 || fields["ok"] != "yes" {
		t.Errorf("fields = %v", fields)
	}
}

func TestParseResponseFieldsLastWins(t *testing.T) {
	fields := yacyproto.ParseResponseFields([]byte("k=first\nk=second\n"))
	if fields["k"] != "second" {
		t.Errorf("k = %q, want second", fields["k"])
	}
}

func TestParseResponseFieldsUsesYaCyTableRules(t *testing.T) {
	fields := yacyproto.ParseResponseFields(
		[]byte("# ignored\n spaced = value \nkey\\=part=line\\nnext\\\\tail\n"),
	)
	if fields["spaced"] != "value" {
		t.Errorf("spaced = %q", fields["spaced"])
	}
	if fields["key=part"] != "line\nnext\\tail" {
		t.Errorf("escaped = %q", fields["key=part"])
	}
}

func TestResponseFieldsEncodeDeterministic(t *testing.T) {
	fields := yacyproto.ResponseFields{"b": "2", "a": "1", "c": "3"}
	want := "a=1\nb=2\nc=3\n"
	if got := fields.Encode(); got != want {
		t.Errorf("Encode() = %q, want %q", got, want)
	}
}

func TestResponseFieldsRoundTrip(t *testing.T) {
	fields := yacyproto.ResponseFields{"version": "1.0", "result": "ok"}
	got := yacyproto.ParseResponseFields([]byte(fields.Encode()))
	if got["version"] != "1.0" || got["result"] != "ok" {
		t.Errorf("round trip = %v", got)
	}
}
