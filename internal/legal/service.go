package legal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) GetActiveDocument(ctx context.Context, documentType domain.LegalDocumentType, language string) (domain.LegalDocument, error) {
	if err := documentType.Validate(); err != nil {
		return domain.LegalDocument{}, err
	}
	if language == "" {
		language = "ru"
	}
	return service.repository.GetActiveDocument(ctx, documentType, language)
}

func (service *Service) ListDocuments(ctx context.Context, documentType *domain.LegalDocumentType, language string) ([]domain.LegalDocument, error) {
	if documentType != nil {
		if err := documentType.Validate(); err != nil {
			return nil, err
		}
	}
	if language == "" {
		language = "ru"
	}
	return service.repository.ListDocuments(ctx, documentType, language)
}

func (service *Service) CreateDocument(ctx context.Context, document domain.LegalDocument, activate bool) (domain.LegalDocument, error) {
	if _, err := domain.NewLegalDocument(document.DocumentType, document.Version, document.Title, document.Content, document.Language); err != nil {
		return domain.LegalDocument{}, err
	}
	if document.Language == "" {
		document.Language = "ru"
	}
	return service.repository.CreateDocument(ctx, document, activate)
}

func (service *Service) ActivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error) {
	return service.repository.ActivateDocument(ctx, documentID)
}

func (service *Service) DeactivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error) {
	return service.repository.DeactivateDocument(ctx, documentID)
}

func (service *Service) AcceptActiveDocument(ctx context.Context, userID uuid.UUID, documentType domain.LegalDocumentType, language string, ip string, userAgent string) (domain.UserDocumentAcceptance, error) {
	document, err := service.GetActiveDocument(ctx, documentType, language)
	if err != nil {
		return domain.UserDocumentAcceptance{}, fmt.Errorf("get active legal document for acceptance: %w", err)
	}
	acceptance := domain.UserDocumentAcceptance{
		UserID:          userID,
		DocumentID:      document.ID,
		DocumentVersion: document.Version,
		AcceptedAt:      time.Now().UTC(),
		IP:              ip,
		UserAgent:       userAgent,
	}
	return service.repository.AcceptDocument(ctx, acceptance)
}
