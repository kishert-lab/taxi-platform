package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeCommission       TransactionType = "commission"
	TransactionTypeDriverIncome     TransactionType = "driver_income"
	TransactionTypeRefund           TransactionType = "refund"
	TransactionTypeManualAdjustment TransactionType = "manual_adjustment"
	TransactionTypeWithdrawal       TransactionType = "withdrawal"
)

const DefaultPlatformCommissionBasisPoints int32 = 100

type CommissionRate struct {
	BasisPoints int32
	Source      string
}

type CommissionContext struct {
	PlatformDefaultBasisPoints int32
	CityBasisPoints            *int32
	TariffBasisPoints          *int32
	TaxiParkBasisPoints        *int32
	DriverBasisPoints          *int32
}

type OrderSettlement struct {
	OrderID                 uuid.UUID
	DriverID                uuid.UUID
	TaxiParkID              *uuid.UUID
	GrossAmount             Money
	CommissionRate          CommissionRate
	CommissionAmount        Money
	NetAmount               Money
	CommissionTransactionID uuid.UUID
	IncomeTransactionID     uuid.UUID
	CreatedAt               time.Time
}

type FinancialTransaction struct {
	ID                    uuid.UUID
	OrderID               *uuid.UUID
	DriverID              *uuid.UUID
	TaxiParkID            *uuid.UUID
	TransactionType       TransactionType
	GrossAmount           Money
	CommissionBasisPoints int32
	CommissionAmount      Money
	NetAmount             Money
	Currency              string
	CreatedAt             time.Time
}

type DriverBalance struct {
	DriverID         uuid.UUID
	AvailableBalance Money
	PendingBalance   Money
	UpdatedAt        time.Time
}

type TaxiParkBalance struct {
	TaxiParkID       uuid.UUID
	AvailableBalance Money
	UpdatedAt        time.Time
}

var (
	ErrInvalidCommissionRate = errors.New("invalid commission rate")
	ErrFinancialOrderIgnored = errors.New("financial order ignored")
)

func ResolveCommissionRate(context CommissionContext) (CommissionRate, error) {
	if context.PlatformDefaultBasisPoints == 0 {
		context.PlatformDefaultBasisPoints = DefaultPlatformCommissionBasisPoints
	}

	rate := CommissionRate{BasisPoints: context.PlatformDefaultBasisPoints, Source: "platform_default"}
	if context.CityBasisPoints != nil {
		rate = CommissionRate{BasisPoints: *context.CityBasisPoints, Source: "city"}
	}
	if context.TariffBasisPoints != nil {
		rate = CommissionRate{BasisPoints: *context.TariffBasisPoints, Source: "tariff"}
	}
	if context.TaxiParkBasisPoints != nil {
		rate = CommissionRate{BasisPoints: *context.TaxiParkBasisPoints, Source: "taxi_park"}
	}
	if context.DriverBasisPoints != nil {
		rate = CommissionRate{BasisPoints: *context.DriverBasisPoints, Source: "driver"}
	}
	if rate.BasisPoints < 0 || rate.BasisPoints > 10000 {
		return CommissionRate{}, ErrInvalidCommissionRate
	}

	return rate, nil
}

func CalculateCommission(grossAmount Money, rate CommissionRate) (Money, Money, error) {
	if grossAmount.Amount < 0 || grossAmount.Currency == "" {
		return Money{}, Money{}, ErrInvalidMoney
	}
	if rate.BasisPoints < 0 || rate.BasisPoints > 10000 {
		return Money{}, Money{}, ErrInvalidCommissionRate
	}

	commissionAmount := (grossAmount.Amount*int64(rate.BasisPoints) + 5000) / 10000
	netAmount := grossAmount.Amount - commissionAmount

	commission, err := NewMoney(commissionAmount, grossAmount.Currency)
	if err != nil {
		return Money{}, Money{}, err
	}
	net, err := NewMoney(netAmount, grossAmount.Currency)
	if err != nil {
		return Money{}, Money{}, err
	}

	return commission, net, nil
}

func (transactionType TransactionType) Validate() error {
	switch transactionType {
	case TransactionTypeCommission,
		TransactionTypeDriverIncome,
		TransactionTypeRefund,
		TransactionTypeManualAdjustment,
		TransactionTypeWithdrawal:
		return nil
	default:
		return errors.New("invalid transaction type")
	}
}
