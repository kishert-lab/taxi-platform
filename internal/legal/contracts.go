package legal

import (
	"context"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type Repository interface {
	GetActiveDocument(ctx context.Context, documentType domain.LegalDocumentType, language string) (domain.LegalDocument, error)
	ListDocuments(ctx context.Context, documentType *domain.LegalDocumentType, language string) ([]domain.LegalDocument, error)
	CreateDocument(ctx context.Context, document domain.LegalDocument, activate bool) (domain.LegalDocument, error)
	ActivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error)
	DeactivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error)
	AcceptDocument(ctx context.Context, acceptance domain.UserDocumentAcceptance) (domain.UserDocumentAcceptance, error)
}
