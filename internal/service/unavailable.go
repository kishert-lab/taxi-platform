package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/common"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/finance"
)

// UnavailableUseCase is used only to expose routes while a concrete application
// service is not wired yet. It returns an explicit 501 through transport mapping.
type UnavailableUseCase struct{}

func NewUnavailableUseCase() *UnavailableUseCase {
	return &UnavailableUseCase{}
}

func (useCase *UnavailableUseCase) StartRegistration(context.Context, auth.StartRegistrationCommand) (auth.StartRegistrationResult, error) {
	return auth.StartRegistrationResult{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) StartLogin(context.Context, dto.AuthLoginRequest) (dto.AuthCodeSentResponse, error) {
	return dto.AuthCodeSentResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) VerifyCode(context.Context, dto.AuthVerifyCodeRequest) (dto.AuthTokenResponse, error) {
	return dto.AuthTokenResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) Refresh(context.Context, dto.RefreshTokenRequest) (dto.AuthTokenResponse, error) {
	return dto.AuthTokenResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) Logout(context.Context, dto.LogoutRequest) error {
	return common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) AuthenticateWebSocket(context.Context, string) (uuid.UUID, domain.UserRole, error) {
	return uuid.Nil, "", common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) CreatePassengerProfile(context.Context, uuid.UUID, dto.PassengerProfileRequest) (dto.PassengerProfileResponse, error) {
	return dto.PassengerProfileResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetPassengerProfile(context.Context, uuid.UUID) (dto.PassengerProfileResponse, error) {
	return dto.PassengerProfileResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UpdatePassengerProfile(context.Context, uuid.UUID, dto.PassengerProfilePatchRequest) (dto.PassengerProfileResponse, error) {
	return dto.PassengerProfileResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UploadPassengerProfilePhoto(context.Context, uuid.UUID, dto.ProfilePhotoUploadRequest) (dto.ProfilePhotoUploadResponse, error) {
	return dto.ProfilePhotoUploadResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) EstimatePassengerOrder(context.Context, uuid.UUID, dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error) {
	return dto.OrderEstimateResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) CreatePassengerOrder(context.Context, uuid.UUID, dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetCurrentPassengerOrder(context.Context, uuid.UUID) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) CurrentForPassenger(context.Context, uuid.UUID) (domain.Order, error) {
	return domain.Order{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListPassengerOrderHistory(context.Context, uuid.UUID) (dto.OrderHistoryResponse, error) {
	return dto.OrderHistoryResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetPassengerOrder(context.Context, uuid.UUID, uuid.UUID) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) CancelPassengerOrder(context.Context, uuid.UUID, uuid.UUID, dto.CancelOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) RatePassengerOrder(context.Context, uuid.UUID, uuid.UUID, dto.RateOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetDriverProfile(context.Context, uuid.UUID) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UpdateDriverProfile(context.Context, uuid.UUID, dto.DriverProfilePatchRequest) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UploadDriverProfilePhoto(context.Context, uuid.UUID, dto.ProfilePhotoUploadRequest) (dto.ProfilePhotoUploadResponse, error) {
	return dto.ProfilePhotoUploadResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) MarkDriverOnline(context.Context, uuid.UUID) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) MarkDriverOffline(context.Context, uuid.UUID) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UpdateDriverLocation(context.Context, uuid.UUID, dto.DriverLocationRequest) error {
	return common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UpdateDriverLocationBatch(context.Context, uuid.UUID, dto.DriverLocationBatchRequest) error {
	return common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetCurrentDriverOrder(context.Context, uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListDriverOrderHistory(context.Context, uuid.UUID) (dto.DriverOrderHistoryResponse, error) {
	return dto.DriverOrderHistoryResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) AcceptDriverOrder(context.Context, uuid.UUID, uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) RejectDriverOrder(context.Context, uuid.UUID, uuid.UUID, dto.RejectOrderRequest) error {
	return common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) MarkDriverArrived(context.Context, uuid.UUID, uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) StartDriverTrip(context.Context, uuid.UUID, uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) CompleteDriverTrip(context.Context, uuid.UUID, uuid.UUID, dto.CompleteOrderRequest) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) RatePassenger(context.Context, uuid.UUID, uuid.UUID, dto.RateOrderRequest) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetDriverBalance(context.Context, uuid.UUID) (domain.DriverBalance, error) {
	return domain.DriverBalance{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListDriverTransactions(context.Context, uuid.UUID, int) ([]domain.FinancialTransaction, error) {
	return nil, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetTaxiParkBalance(context.Context, uuid.UUID) (domain.TaxiParkBalance, error) {
	return domain.TaxiParkBalance{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListTaxiParkDrivers(context.Context, uuid.UUID, int) ([]finance.TaxiParkDriver, error) {
	return nil, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListTaxiParkOrders(context.Context, uuid.UUID, int) ([]finance.TaxiParkOrder, error) {
	return nil, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListTaxiParkTransactions(context.Context, uuid.UUID, int) ([]domain.FinancialTransaction, error) {
	return nil, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetAdminOverview(context.Context, time.Time, time.Time) (finance.AdminOverview, error) {
	return finance.AdminOverview{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetSettings(context.Context, uuid.UUID) (domain.TaxiParkSettings, error) {
	return domain.TaxiParkSettings{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UpdateSettings(context.Context, uuid.UUID, dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error) {
	return domain.TaxiParkSettings{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListTariffs(context.Context, uuid.UUID) ([]domain.TaxiParkTariff, error) {
	return nil, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) CreateTariff(context.Context, uuid.UUID, dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error) {
	return domain.TaxiParkTariff{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) UpdateTariff(context.Context, uuid.UUID, uuid.UUID, dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error) {
	return domain.TaxiParkTariff{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) GetActiveDocument(context.Context, domain.LegalDocumentType, string) (domain.LegalDocument, error) {
	return domain.LegalDocument{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ListDocuments(context.Context, *domain.LegalDocumentType, string) ([]domain.LegalDocument, error) {
	return nil, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) CreateDocument(context.Context, domain.LegalDocument, bool) (domain.LegalDocument, error) {
	return domain.LegalDocument{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) ActivateDocument(context.Context, uuid.UUID) (domain.LegalDocument, error) {
	return domain.LegalDocument{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) DeactivateDocument(context.Context, uuid.UUID) (domain.LegalDocument, error) {
	return domain.LegalDocument{}, common.ErrNotImplemented
}

func (useCase *UnavailableUseCase) AcceptActiveDocument(context.Context, uuid.UUID, domain.LegalDocumentType, string, string, string) (domain.UserDocumentAcceptance, error) {
	return domain.UserDocumentAcceptance{}, common.ErrNotImplemented
}
