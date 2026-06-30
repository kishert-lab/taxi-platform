package passenger

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
)

type AddressSearcher interface {
	Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error)
	ResolveCityByCoordinates(ctx context.Context, coordinates geodomain.Coordinates) (geoservice.CityContext, bool, error)
}

type AddressSearchUseCase interface {
	SearchPassengerAddresses(
		ctx context.Context,
		passengerID uuid.UUID,
		query string,
		cityID *uuid.UUID,
		focusLatitude *float64,
		focusLongitude *float64,
		limit int,
	) ([]geodomain.SearchResult, error)
}

type AddressSearchService struct {
	searcher AddressSearcher
}

func NewAddressSearchService(searcher AddressSearcher) *AddressSearchService {
	return &AddressSearchService{searcher: searcher}
}

func (service *AddressSearchService) SearchPassengerAddresses(
	ctx context.Context,
	passengerID uuid.UUID,
	query string,
	cityID *uuid.UUID,
	focusLatitude *float64,
	focusLongitude *float64,
	limit int,
) ([]geodomain.SearchResult, error) {
	request := geodomain.SearchRequest{
		Query:       query,
		CityID:      cityID,
		ActorUserID: &passengerID,
		ActorRole:   string(domain.UserRolePassenger),
		Limit:       limit,
	}

	if focusLatitude != nil || focusLongitude != nil {
		if focusLatitude == nil || focusLongitude == nil {
			return nil, geodomain.ErrInvalidCoordinates
		}
		coordinates, err := geodomain.NewCoordinates(*focusLatitude, *focusLongitude)
		if err != nil {
			return nil, fmt.Errorf("build passenger geocoder focus: %w", err)
		}
		request.Focus = &coordinates
		if request.CityID == nil {
			cityContext, found, err := service.searcher.ResolveCityByCoordinates(ctx, coordinates)
			if err != nil {
				return nil, fmt.Errorf("resolve passenger city by coordinates: %w", err)
			}
			if found {
				request.CityID = &cityContext.CityID
			}
		}
	}

	results, err := service.searcher.Search(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("search passenger addresses: %w", err)
	}
	return results, nil
}
