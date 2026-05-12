package domain

import "errors"

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

var ErrInvalidCoordinates = errors.New("invalid coordinates")

func NewCoordinates(latitude float64, longitude float64) (Coordinates, error) {
	if latitude < -90 || latitude > 90 {
		return Coordinates{}, ErrInvalidCoordinates
	}
	if longitude < -180 || longitude > 180 {
		return Coordinates{}, ErrInvalidCoordinates
	}

	return Coordinates{Latitude: latitude, Longitude: longitude}, nil
}
