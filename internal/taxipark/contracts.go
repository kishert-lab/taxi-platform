package taxipark

import (
	"context"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type Repository interface {
	GetSettingsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error)
	UpdateSettingsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error)
	ListTariffsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) ([]domain.TaxiParkTariff, error)
	CreateTariffByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error)
	UpdateTariffByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, tariffID uuid.UUID, request dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error)
}
