package taxipark

import (
	"context"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) GetSettings(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error) {
	return service.repository.GetSettingsByOwnerUserID(ctx, ownerUserID)
}

func (service *Service) UpdateSettings(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error) {
	return service.repository.UpdateSettingsByOwnerUserID(ctx, ownerUserID, request)
}

func (service *Service) ListTariffs(ctx context.Context, ownerUserID uuid.UUID) ([]domain.TaxiParkTariff, error) {
	return service.repository.ListTariffsByOwnerUserID(ctx, ownerUserID)
}

func (service *Service) CreateTariff(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error) {
	return service.repository.CreateTariffByOwnerUserID(ctx, ownerUserID, request)
}

func (service *Service) UpdateTariff(ctx context.Context, ownerUserID uuid.UUID, tariffID uuid.UUID, request dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error) {
	return service.repository.UpdateTariffByOwnerUserID(ctx, ownerUserID, tariffID, request)
}
