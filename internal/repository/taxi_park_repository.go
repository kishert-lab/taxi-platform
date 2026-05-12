package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresTaxiParkRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTaxiParkRepository(pool *pgxpool.Pool) *PostgresTaxiParkRepository {
	return &PostgresTaxiParkRepository{pool: pool}
}

func (repository *PostgresTaxiParkRepository) CreateTaxiPark(ctx context.Context, taxiPark domain.TaxiPark) (domain.TaxiPark, error) {
	const query = `
		INSERT INTO taxi_parks (
			owner_user_id,
			city_id,
			name,
			legal_name,
			tax_id,
			contact_phone,
			contact_email,
			is_verified,
			verification_status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id,
			owner_user_id,
			city_id,
			name,
			legal_name,
			tax_id,
			contact_phone,
			contact_email,
			is_verified,
			verification_status,
			created_at,
			updated_at,
			deleted_at`

	createdTaxiPark, err := scanTaxiPark(repository.pool.QueryRow(
		ctx,
		query,
		taxiPark.OwnerUserID,
		taxiPark.CityID,
		taxiPark.Name,
		nullableString(taxiPark.LegalName),
		nullableString(taxiPark.TaxID),
		taxiPark.ContactPhone,
		taxiPark.ContactEmail,
		taxiPark.IsVerified,
		taxiPark.VerificationStatus,
	))
	if err != nil {
		return domain.TaxiPark{}, fmt.Errorf("insert taxi park: %w", err)
	}

	return createdTaxiPark, nil
}

func (repository *PostgresTaxiParkRepository) GetTaxiParkByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiPark, error) {
	const query = `
		SELECT
			id,
			owner_user_id,
			city_id,
			name,
			legal_name,
			tax_id,
			contact_phone,
			contact_email,
			is_verified,
			verification_status,
			created_at,
			updated_at,
			deleted_at
		FROM taxi_parks
		WHERE owner_user_id = $1 AND deleted_at IS NULL`

	taxiPark, err := scanTaxiPark(repository.pool.QueryRow(ctx, query, ownerUserID))
	if err != nil {
		return domain.TaxiPark{}, fmt.Errorf("select taxi park by owner user id: %w", err)
	}

	return taxiPark, nil
}

func scanTaxiPark(row pgx.Row) (domain.TaxiPark, error) {
	var taxiPark domain.TaxiPark
	var legalName pgtype.Text
	var taxID pgtype.Text
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&taxiPark.ID,
		&taxiPark.OwnerUserID,
		&taxiPark.CityID,
		&taxiPark.Name,
		&legalName,
		&taxID,
		&taxiPark.ContactPhone,
		&taxiPark.ContactEmail,
		&taxiPark.IsVerified,
		&taxiPark.VerificationStatus,
		&taxiPark.CreatedAt,
		&taxiPark.UpdatedAt,
		&deletedAt,
	); err != nil {
		return domain.TaxiPark{}, err
	}

	taxiPark.LegalName = legalName.String
	taxiPark.TaxID = taxID.String
	if deletedAt.Valid {
		taxiPark.DeletedAt = &deletedAt.Time
	}

	return taxiPark, nil
}
