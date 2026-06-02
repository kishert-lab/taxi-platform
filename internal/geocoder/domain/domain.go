// Package domain contains pure geocoder entities and validation rules.
package domain

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Provider string

const (
	ProviderLocal  Provider = "local"
	ProviderPelias Provider = "pelias"
	ProviderYandex Provider = "yandex"
	ProviderDaData Provider = "dadata"
)

type TrustLevel string

const (
	TrustLevelConfirmed TrustLevel = "confirmed"
	TrustLevelTrusted   TrustLevel = "trusted"
	TrustLevelRejected  TrustLevel = "rejected"
)

type PointSource string

const (
	PointSourceUserConfirmed       PointSource = "user_confirmed"
	PointSourceDriverConfirmed     PointSource = "driver_confirmed"
	PointSourceDispatcherConfirmed PointSource = "dispatcher_confirmed"
	PointSourceAdmin               PointSource = "admin"
)

type ConfirmationAction string

const (
	ConfirmationActionConfirm ConfirmationAction = "confirm"
	ConfirmationActionReject  ConfirmationAction = "reject"
)

var (
	ErrInvalidQuery        = errors.New("invalid geocoder query")
	ErrInvalidCoordinates  = errors.New("invalid geocoder coordinates")
	ErrInvalidConfidence   = errors.New("invalid geocoder confidence")
	ErrPointNotFound       = errors.New("local geo point not found")
	ErrExternalUnavailable = errors.New("external geocoder unavailable")
)

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func NewCoordinates(latitude float64, longitude float64) (Coordinates, error) {
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 {
		return Coordinates{}, ErrInvalidCoordinates
	}
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		return Coordinates{}, ErrInvalidCoordinates
	}
	return Coordinates{Latitude: latitude, Longitude: longitude}, nil
}

type SearchRequest struct {
	Query       string
	CityID      *uuid.UUID
	Focus       *Coordinates
	ActorUserID *uuid.UUID
	ActorRole   string
	Limit       int
	RequestedAt time.Time
}

type SearchResult struct {
	ID              string
	LocalPointID    *uuid.UUID
	Provider        Provider
	Name            string
	Address         string
	CityID          *uuid.UUID
	Coordinates     Coordinates
	Confidence      float64
	TrustLevel      TrustLevel
	ExternalPlaceID string
	ExpiresAt       *time.Time
}

type LocalGeoPoint struct {
	ID                uuid.UUID
	CityID            uuid.UUID
	Name              string
	NormalizedName    string
	Address           string
	Coordinates       Coordinates
	Source            PointSource
	ExternalProvider  string
	ExternalPlaceID   string
	Confidence        float64
	TrustLevel        TrustLevel
	ConfirmationCount int
	RejectCount       int
	ApprovedBy        *uuid.UUID
	ApprovedAt        *time.Time
	RejectedBy        *uuid.UUID
	RejectedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Confirmation struct {
	PointID   uuid.UUID
	UserID    *uuid.UUID
	ActorRole string
	Action    ConfirmationAction
	Source    PointSource
	Address   string
	Location  Coordinates
	Comment   string
	IP        string
	UserAgent string
	CreatedAt time.Time
}

func NormalizeQuery(query string) string {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	return strings.Join(fields, " ")
}

func ValidateConfidence(confidence float64) error {
	if math.IsNaN(confidence) || math.IsInf(confidence, 0) || confidence < 0 || confidence > 1 {
		return ErrInvalidConfidence
	}
	return nil
}
