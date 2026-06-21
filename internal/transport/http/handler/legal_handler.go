package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type LegalUseCase interface {
	GetActiveDocument(ctx context.Context, documentType domain.LegalDocumentType, language string) (domain.LegalDocument, error)
	ListDocuments(ctx context.Context, documentType *domain.LegalDocumentType, language string) ([]domain.LegalDocument, error)
	CreateDocument(ctx context.Context, document domain.LegalDocument, activate bool) (domain.LegalDocument, error)
	ActivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error)
	DeactivateDocument(ctx context.Context, documentID uuid.UUID) (domain.LegalDocument, error)
	AcceptActiveDocument(ctx context.Context, userID uuid.UUID, documentType domain.LegalDocumentType, language string, ip string, userAgent string) (domain.UserDocumentAcceptance, error)
}

type LegalHandler struct {
	useCase LegalUseCase
}

func NewLegalHandler(useCase LegalUseCase) *LegalHandler {
	return &LegalHandler{useCase: useCase}
}

func (handler *LegalHandler) RegisterRoutes(router gin.IRouter) {
	public := router.Group("/public/legal")
	public.GET("/privacy-policy", handler.PublicPrivacyPolicy)
	public.GET("/terms", handler.PublicTerms)
	public.GET("/consent", handler.PublicConsent)
	public.GET("/documents/:document_type", handler.PublicDocumentByType)

	admin := router.Group("/admin/legal", middleware.RequireRole(domain.UserRoleAdmin))
	admin.GET("/documents", handler.AdminListDocuments)
	admin.POST("/documents", handler.AdminCreateDocument)
	admin.POST("/documents/:id/activate", handler.AdminActivateDocument)
	admin.POST("/documents/:id/deactivate", handler.AdminDeactivateDocument)
}

// PublicPrivacyPolicy godoc
// @Summary Get active privacy policy
// @Description Returns the currently active immutable privacy policy version from legal_documents.
// @Tags public-legal
// @Produce json
// @Param language query string false "Language" default(ru)
// @Success 200 {object} LegalDocumentSuccessResponse
// @Failure 404 {object} response.Error
// @Router /public/legal/privacy-policy [get]
func (handler *LegalHandler) PublicPrivacyPolicy(context *gin.Context) {
	handler.publicDocument(context, domain.LegalDocumentPrivacyPolicy)
}

// PublicTerms godoc
// @Summary Get active terms of service
// @Description Returns the currently active immutable terms of service version from legal_documents.
// @Tags public-legal
// @Produce json
// @Param language query string false "Language" default(ru)
// @Success 200 {object} LegalDocumentSuccessResponse
// @Failure 404 {object} response.Error
// @Router /public/legal/terms [get]
func (handler *LegalHandler) PublicTerms(context *gin.Context) {
	handler.publicDocument(context, domain.LegalDocumentTermsOfService)
}

// PublicConsent godoc
// @Summary Get active personal data consent
// @Description Returns the currently active immutable personal data consent version from legal_documents.
// @Tags public-legal
// @Produce json
// @Param language query string false "Language" default(ru)
// @Success 200 {object} LegalDocumentSuccessResponse
// @Failure 404 {object} response.Error
// @Router /public/legal/consent [get]
func (handler *LegalHandler) PublicConsent(context *gin.Context) {
	handler.publicDocument(context, domain.LegalDocumentConsentPersonalData)
}

// PublicDocumentByType godoc
// @Summary Get active legal document by type
// @Description Returns the active immutable legal document version for the requested document type.
// @Tags public-legal
// @Produce json
// @Param document_type path string true "Document type"
// @Param language query string false "Language" default(ru)
// @Success 200 {object} LegalDocumentSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /public/legal/documents/{document_type} [get]
func (handler *LegalHandler) PublicDocumentByType(context *gin.Context) {
	documentType := domain.LegalDocumentType(context.Param("document_type"))
	handler.publicDocument(context, documentType)
}

func (handler *LegalHandler) publicDocument(context *gin.Context, documentType domain.LegalDocumentType) {
	document, err := handler.useCase.GetActiveDocument(context.Request.Context(), documentType, context.DefaultQuery("language", "ru"))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.LegalDocumentFromDomain(document))
}

// AdminListDocuments godoc
// @Summary List legal document versions
// @Description Lists immutable legal document versions filtered by type and language.
// @Tags admin-legal
// @Produce json
// @Security BearerAuth
// @Param document_type query string false "Document type"
// @Param language query string false "Language" default(ru)
// @Success 200 {object} LegalDocumentsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/legal/documents [get]
func (handler *LegalHandler) AdminListDocuments(context *gin.Context) {
	var documentType *domain.LegalDocumentType
	if value := context.Query("document_type"); value != "" {
		typedValue := domain.LegalDocumentType(value)
		documentType = &typedValue
	}
	documents, err := handler.useCase.ListDocuments(context.Request.Context(), documentType, context.DefaultQuery("language", "ru"))
	if err != nil {
		failByError(context, err)
		return
	}
	responseBody := dto.LegalDocumentsResponse{Documents: make([]dto.LegalDocumentResponse, 0, len(documents))}
	for _, document := range documents {
		responseBody.Documents = append(responseBody.Documents, dto.LegalDocumentFromDomain(document))
	}
	response.OK(context, responseBody)
}

// AdminCreateDocument godoc
// @Summary Create new immutable legal document version
// @Description Creates a new legal document row. When activate=true, older versions of the same type and language become inactive.
// @Tags admin-legal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.LegalDocumentRequest true "Legal document"
// @Success 201 {object} LegalDocumentSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/legal/documents [post]
func (handler *LegalHandler) AdminCreateDocument(context *gin.Context) {
	var request dto.LegalDocumentRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid legal document request")
		return
	}
	document := domain.LegalDocument{
		DocumentType: request.DocumentType,
		Version:      request.Version,
		Title:        request.Title,
		Content:      request.Content,
		Language:     request.Language,
	}
	createdDocument, err := handler.useCase.CreateDocument(context.Request.Context(), document, request.Activate)
	if err != nil {
		failByError(context, err)
		return
	}
	response.Created(context, dto.LegalDocumentFromDomain(createdDocument))
}

// AdminActivateDocument godoc
// @Summary Activate legal document version
// @Description Activates the selected legal document version and deactivates other versions of the same type and language.
// @Tags admin-legal
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID"
// @Success 200 {object} LegalDocumentSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/legal/documents/{id}/activate [post]
func (handler *LegalHandler) AdminActivateDocument(context *gin.Context) {
	handler.changeDocumentActivation(context, true)
}

// AdminDeactivateDocument godoc
// @Summary Deactivate legal document version
// @Description Deactivates the selected legal document version.
// @Tags admin-legal
// @Produce json
// @Security BearerAuth
// @Param id path string true "Document ID"
// @Success 200 {object} LegalDocumentSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/legal/documents/{id}/deactivate [post]
func (handler *LegalHandler) AdminDeactivateDocument(context *gin.Context) {
	handler.changeDocumentActivation(context, false)
}

func (handler *LegalHandler) changeDocumentActivation(context *gin.Context, activate bool) {
	documentID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid document id")
		return
	}
	var document domain.LegalDocument
	if activate {
		document, err = handler.useCase.ActivateDocument(context.Request.Context(), documentID)
	} else {
		document, err = handler.useCase.DeactivateDocument(context.Request.Context(), documentID)
	}
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.LegalDocumentFromDomain(document))
}
