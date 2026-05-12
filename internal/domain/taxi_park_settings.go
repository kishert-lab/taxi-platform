package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaxiParkSettings struct {
	ID                      uuid.UUID
	TaxiParkID              uuid.UUID
	DisplayName             string
	ShortName               string
	SupportPhone            string
	SupportEmail            string
	LegalName               string
	LegalAddress            string
	INN                     string
	OGRN                    string
	Website                 string
	LogoURL                 string
	PrimaryColor            string
	SecondaryColor          string
	CommissionBasisPoints   *int32
	MinimumOrderPrice       Money
	CancellationTimeoutSec  int
	DriverArrivalTimeoutSec int
	AllowCashPayment        bool
	AllowCardPayment        bool
	AllowTransferPayment    bool
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
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
