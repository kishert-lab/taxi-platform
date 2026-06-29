package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaxiParkSettings struct {
	ID                                uuid.UUID
	TaxiParkID                        uuid.UUID
	CityID                            uuid.UUID
	CityName                          string
	CityRegion                        string
	CityCountryCode                   string
	CityTimezone                      string
	CityCenter                        Coordinates
	DisplayName                       string
	ShortName                         string
	SupportPhone                      string
	SupportEmail                      string
	LegalName                         string
	LegalAddress                      string
	INN                               string
	OGRN                              string
	Website                           string
	LogoURL                           string
	PrimaryColor                      string
	SecondaryColor                    string
	CommissionBasisPoints             *int32
	MinimumOrderPrice                 Money
	CancellationTimeoutSec            int
	DriverArrivalTimeoutSec           int
	AllowCashPayment                  bool
	AllowCardPayment                  bool
	AllowTransferPayment              bool
	DispatchInitialRadiusMeters       int
	DispatchMaxRadiusMeters           int
	DispatchRadiusStepMeters          int
	DispatchRadiusAttemptsMeters      []int
	DispatchMaxDriversPerOffer        int
	DispatchDriverLocationMaxAgeSec   int
	DispatchOfferTTLSec               int
	DispatchAcceptLockTTLSec          int
	DispatchWorkerPollTimeoutSec      int
	DispatchRecoveryIntervalSec       int
	ScheduledOrdersEnabled            bool
	ScheduledMinBeforeMinutes         int
	ScheduledActivationBeforeMinutes  int
	ScheduledExpireAfterMinutes       int
	AllowScheduledDriverPreassignment bool
	IsActive                          bool
	CreatedAt                         time.Time
	UpdatedAt                         time.Time
}

type TaxiParkTariff struct {
	ID             uuid.UUID
	TaxiParkID     uuid.UUID
	Name           string
	Description    string
	BasePrice      Money
	PricePerKM     Money
	PricePerMinute Money
	MinimumPrice   Money
	FixedRoutes    []byte
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
