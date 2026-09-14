package processenvironmentlease_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
)

func TestGrantCarriesEveryGrantedName(t *testing.T) {
	frame := []byte(`{"protocol_version":1,"process_environment":` +
		`{"YACY_ADVERTISE_HOST":"name.localhost.run","YACY_ADVERTISE_PORT":"80"}}` + "\n")

	grant, err := processenvironmentlease.GrantFrom(frame)
	if err != nil {
		t.Fatalf("GrantFrom: %v", err)
	}

	want := map[string]string{
		"YACY_ADVERTISE_HOST": "name.localhost.run",
		"YACY_ADVERTISE_PORT": "80",
	}
	if len(grant.ProcessEnvironment) != len(want) {
		t.Fatalf("ProcessEnvironment = %v, want %v", grant.ProcessEnvironment, want)
	}
	for name, value := range want {
		if grant.ProcessEnvironment[name] != value {
			t.Errorf("ProcessEnvironment[%s] = %q, want %q",
				name, grant.ProcessEnvironment[name], value)
		}
	}
}

func TestGrantAcceptsAnEmptyProcessEnvironment(t *testing.T) {
	grant, err := processenvironmentlease.GrantFrom(
		[]byte(`{"protocol_version":1,"process_environment":{}}`),
	)
	if err != nil {
		t.Fatalf("GrantFrom: %v", err)
	}
	if len(grant.ProcessEnvironment) != 0 {
		t.Errorf("ProcessEnvironment = %v, want empty", grant.ProcessEnvironment)
	}
}

func TestGrantRejectsABadFrame(t *testing.T) {
	oversizedValue := strings.Repeat("v", processenvironmentlease.MaximumFrameByte)

	for name, frame := range map[string]string{
		"unsupported version": `{"protocol_version":2,"process_environment":{}}`,
		"absent version":      `{"process_environment":{}}`,
		"absent environment":  `{"protocol_version":1}`,
		"unknown name":        `{"protocol_version":1,"process_environment":{},"expiry":"soon"}`,
		"duplicate name":      `{"protocol_version":1,"protocol_version":1,"process_environment":{}}`,
		"trailing value": `{"protocol_version":1,"process_environment":{}} ` +
			`{"protocol_version":1,"process_environment":{}}`,
		"data after the newline": "{\"protocol_version\":1,\"process_environment\":{}}\n!",
		"empty environment name": `{"protocol_version":1,"process_environment":{"":"v"}}`,
		"equals in name":         `{"protocol_version":1,"process_environment":{"A=B":"v"}}`,
		"null byte in name":      `{"protocol_version":1,"process_environment":{"A\u0000B":"v"}}`,
		"null byte in value":     `{"protocol_version":1,"process_environment":{"A":"v\u0000"}}`,
		"value is not text":      `{"protocol_version":1,"process_environment":{"A":80}}`,
		"frame is not JSON":      `not json`,
		"oversized frame": fmt.Sprintf(
			`{"protocol_version":1,"process_environment":{"A":%q}}`, oversizedValue,
		),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := processenvironmentlease.GrantFrom([]byte(frame)); err == nil {
				t.Fatalf("GrantFrom accepted %s", name)
			} else if !errors.Is(err, processenvironmentlease.ErrBadGrant) {
				t.Errorf("GrantFrom error = %v, want ErrBadGrant", err)
			}
		})
	}
}

func TestFrameRoundTripsAGrant(t *testing.T) {
	granted := processenvironmentlease.Grant{
		ProcessEnvironment: map[string]string{"YACY_TRUSTED_PROXIES": "172.20.0.3/32"},
	}

	frame, err := processenvironmentlease.FrameOf(granted)
	if err != nil {
		t.Fatalf("FrameOf: %v", err)
	}
	if !strings.HasSuffix(string(frame), "\n") {
		t.Errorf("FrameOf = %q, want a trailing newline", frame)
	}

	returned, err := processenvironmentlease.GrantFrom(frame)
	if err != nil {
		t.Fatalf("GrantFrom: %v", err)
	}
	if returned.ProcessEnvironment["YACY_TRUSTED_PROXIES"] != "172.20.0.3/32" {
		t.Errorf("ProcessEnvironment = %v, want the granted value", returned.ProcessEnvironment)
	}
}

func TestFrameOfAnEmptyGrantIsReadable(t *testing.T) {
	frame, err := processenvironmentlease.FrameOf(processenvironmentlease.Grant{})
	if err != nil {
		t.Fatalf("FrameOf: %v", err)
	}
	if _, err := processenvironmentlease.GrantFrom(frame); err != nil {
		t.Errorf("GrantFrom: %v", err)
	}
}

func TestFrameOfRejectsABadProcessEnvironment(t *testing.T) {
	for name, processEnvironment := range map[string]map[string]string{
		"empty name":         {"": "v"},
		"equals in name":     {"A=B": "v"},
		"null byte in name":  {"A\x00B": "v"},
		"null byte in value": {"A": "v\x00"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := processenvironmentlease.FrameOf(
				processenvironmentlease.Grant{ProcessEnvironment: processEnvironment},
			)
			if err == nil {
				t.Fatalf("FrameOf accepted %s", name)
			}
			if !errors.Is(err, processenvironmentlease.ErrBadGrant) {
				t.Errorf("FrameOf error = %v, want ErrBadGrant", err)
			}
		})
	}
}

func TestFrameOfRejectsAnOversizedProcessEnvironment(t *testing.T) {
	granted := processenvironmentlease.Grant{
		ProcessEnvironment: map[string]string{
			"A": strings.Repeat("v", processenvironmentlease.MaximumFrameByte),
		},
	}

	if _, err := processenvironmentlease.FrameOf(granted); err == nil {
		t.Fatal("FrameOf accepted an oversized process environment")
	}
}

func TestLeasedProcessEnvironmentPrefersGrantedValues(t *testing.T) {
	granted := processenvironmentlease.Grant{
		ProcessEnvironment: map[string]string{
			"YACY_ADVERTISE_HOST": "name.localhost.run",
			"YACY_ADVERTISE_PORT": "80",
		},
	}

	leased := granted.LeasedProcessEnvironmentFrom([]string{
		"YACY_ADVERTISE_HOST=192.0.2.1",
		"YACY_DATA_DIR=/data",
	})

	want := []string{
		"YACY_ADVERTISE_HOST=name.localhost.run",
		"YACY_ADVERTISE_PORT=80",
		"YACY_DATA_DIR=/data",
	}
	if !slices.Equal(leased, want) {
		t.Errorf("LeasedProcessEnvironmentFrom = %v, want %v", leased, want)
	}
}

func TestLeasedProcessEnvironmentKeepsOneEntryPerName(t *testing.T) {
	leased := processenvironmentlease.Grant{}.LeasedProcessEnvironmentFrom([]string{
		"YACY_DATA_DIR=/first",
		"YACY_DATA_DIR=/last",
		"no equals sign",
	})

	if !slices.Equal(leased, []string{"YACY_DATA_DIR=/last"}) {
		t.Errorf("LeasedProcessEnvironmentFrom = %v, want the last value alone", leased)
	}
}
