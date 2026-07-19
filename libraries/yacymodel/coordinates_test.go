package yacymodel_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestNewCoordinatesAcceptsPointsOnEarth(t *testing.T) {
	got, err := yacymodel.NewCoordinates(52.52, 13.405)
	if err != nil {
		t.Fatalf("NewCoordinates: %v", err)
	}
	want := yacymodel.Coordinates{Latitude: 52.52, Longitude: 13.405}
	if got != want {
		t.Errorf("NewCoordinates = %+v, want %+v", got, want)
	}
}

func TestNewCoordinatesRejectsOutOfRange(t *testing.T) {
	for _, c := range []struct {
		name                string
		latitude, longitude float64
	}{
		{"latitude too high", 90.1, 0},
		{"latitude too low", -90.1, 0},
		{"longitude too high", 0, 180.1},
		{"longitude too low", 0, -180.1},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := yacymodel.NewCoordinates(
				c.latitude, c.longitude,
			); !errors.Is(err, yacymodel.ErrCoordinatesOutOfRange) {
				t.Errorf("err = %v, want ErrCoordinatesOutOfRange", err)
			}
		})
	}
}

func TestCoordinatesIsZero(t *testing.T) {
	if !(yacymodel.Coordinates{}).IsZero() {
		t.Error("the origin should report itself absent")
	}
	if (yacymodel.Coordinates{Latitude: 52.52}).IsZero() {
		t.Error("a real point should not report itself absent")
	}
}
