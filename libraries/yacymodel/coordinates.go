package yacymodel

import (
	"errors"
	"fmt"
)

const (
	latitudeLimit  = 90
	longitudeLimit = 180
)

var ErrCoordinatesOutOfRange = errors.New("coordinates out of range")

// Coordinates is a point on the Earth's surface in decimal degrees.
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func (c Coordinates) IsZero() bool {
	return c == Coordinates{}
}

func NewCoordinates(latitude, longitude float64) (Coordinates, error) {
	if latitude < -latitudeLimit || latitude > latitudeLimit {
		return Coordinates{}, fmt.Errorf(
			"%w: latitude %v", ErrCoordinatesOutOfRange, latitude,
		)
	}
	if longitude < -longitudeLimit || longitude > longitudeLimit {
		return Coordinates{}, fmt.Errorf(
			"%w: longitude %v", ErrCoordinatesOutOfRange, longitude,
		)
	}

	return Coordinates{Latitude: latitude, Longitude: longitude}, nil
}
