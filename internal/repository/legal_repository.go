package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresLegalRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresLegalRepository(pool *pgxpool.Pool) *PostgresLegalRepository {
	return &PostgresLegalRepository{pool: pool}
}

func (repository *PostgresLegalRepository) GetActiveDocument(ctx context.Context, documentType domain.LegalDocumentType, language string) (domain.LegalDocument, error) {
	document, err := scanLegalDocument(repository.pool.QueryRow(ctx, `SELECT `+legalDocumentColumns+`
		FROM legal_documents
		WHERE document_type = $1 AND language = $2 AND is_active = true
		ORDER BY created_at DESC
		LIMIT 1`, documentType, language))
	if err != nil {
		return domain.LegalDocument{}, fmt.Errorf("select active legal document: %w", err)
	}
	return document, nil
}

func (repository *PostgresLegalRepository) ListDocuments(ctx context.Context, documentType *domain.LegalDocumentType, language string) ([]domain.LegalDocument, error) {
	rows, err := repository.pool.Query(ctx, `SELECT `+legalDocumentColumns+`
		FROM legal_documents
		WHERE ($1::text IS NULL OR document_type = $1)
		  AND language = $2
		ORDER BY document_type, created_at DESC`, documentType, language)
	if err != nil {
		return nil, fmt.Errorf("select legal documents: %w", err)
	}
	defer rows.Close()

	documents := make([]domain.LegalDocument, 0)
	for rows.Next() {
		document, err := scanLegalDocument(rows)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legal documents: %w", err)
	}
	return documents, nil
}

func (repository *PostgresLegalRepository) CreateDocument(ctx context.Context, document domain.LegalDocument, activate bool) (domain.LegalDocument, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.LegalDocument{}, fmt.Errorf("begin legal document transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	if activate {
		if _, err := tx.Exec(ctx, `UPDATE legal_documents SET is_active = false WHERE document_type = $1 AND language = $2`, document.DocumentType, document.Language); err != nil {
			return domain.LegalDocument{}, fmt.Errorf("deactivate previous legal documents: %w", err)
		}
	}

	createdDocument, err := scanLegalDocument(tx.QueryRow(ctx, `INSERT INTO legal_documents (
			document_type, version, title, content, language, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+legalDocumentColumns,
		document.DocumentType,
		document.Version,
		document.Title,
		document.Content,
		document.Language,
		activate,
	))
	if err != nil {
		return domain.LegalDocument{}, fmt.Errorf("insert legal document: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.LegalDocument{}, fmt.Errorf("commit legal document transaction: %w", err)
	}
	return createdDocument, nil
}

func (repository *PostgresLegalRepository) ActivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error) {
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.LegalDocument{}, fmt.Errorf("begin legal activation transaction: %w", err)
	}
	defer rollbackTx(ctx, tx)

	var documentType domain.LegalDocumentType
	var language string
	if err := tx.QueryRow(ctx, `SELECT document_type, language FROM legal_documents WHERE id = $1`, documentID).Scan(&documentType, &language); err != nil {
		return domain.LegalDocument{}, fmt.Errorf("select legal document for activation: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE legal_documents SET is_active = false WHERE document_type = $1 AND language = $2`, documentType, language); err != nil {
		return domain.LegalDocument{}, fmt.Errorf("deactivate legal document versions: %w", err)
	}
	document, err := scanLegalDocument(tx.QueryRow(ctx, `UPDATE legal_documents SET is_active = true WHERE id = $1 RETURNING `+legalDocumentColumns, documentID))
	if err != nil {
		return domain.LegalDocument{}, fmt.Errorf("activate legal document: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.LegalDocument{}, fmt.Errorf("commit legal activation transaction: %w", err)
	}
	return document, nil
}

func (repository *PostgresLegalRepository) DeactivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error) {
	document, err := scanLegalDocument(repository.pool.QueryRow(ctx, `UPDATE legal_documents SET is_active = false WHERE id = $1 RETURNING `+legalDocumentColumns, documentID))
	if err != nil {
		return domain.LegalDocument{}, fmt.Errorf("deactivate legal document: %w", err)
	}
	return document, nil
}

func (repository *PostgresLegalRepository) AcceptDocument(ctx context.Context, acceptance domain.UserDocumentAcceptance) (domain.UserDocumentAcceptance, error) {
	createdAcceptance, err := scanUserDocumentAcceptance(repository.pool.QueryRow(ctx, `INSERT INTO user_document_acceptance (
			user_id, document_id, document_version, accepted_at, ip, user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, document_id) DO NOTHING
		RETURNING id, user_id, document_id, document_version, accepted_at, ip, user_agent`,
		acceptance.UserID,
		acceptance.DocumentID,
		acceptance.DocumentVersion,
		acceptance.AcceptedAt,
		nullableString(acceptance.IP),
		nullableString(acceptance.UserAgent),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return acceptance, nil
		}
		return domain.UserDocumentAcceptance{}, fmt.Errorf("insert user document acceptance: %w", err)
	}
	return createdAcceptance, nil
}

const legalDocumentColumns = `id, document_type, version, title, content, language, is_active, created_at`

func scanLegalDocument(row pgx.Row) (domain.LegalDocument, error) {
	var document domain.LegalDocument
	if err := row.Scan(&document.ID, &document.DocumentType, &document.Version, &document.Title, &document.Content, &document.Language, &document.IsActive, &document.CreatedAt); err != nil {
		return domain.LegalDocument{}, err
	}
	return document, nil
}

func scanUserDocumentAcceptance(row pgx.Row) (domain.UserDocumentAcceptance, error) {
	var acceptance domain.UserDocumentAcceptance
	var ip pgtype.Text
	var userAgent pgtype.Text
	if err := row.Scan(&acceptance.ID, &acceptance.UserID, &acceptance.DocumentID, &acceptance.DocumentVersion, &acceptance.AcceptedAt, &ip, &userAgent); err != nil {
		return domain.UserDocumentAcceptance{}, err
	}
	acceptance.IP = ip.String
	acceptance.UserAgent = userAgent.String
	return acceptance, nil
}
