package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	passengerapp "github.com/kishert-lab/taxi-platform/internal/passenger"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

const PassengerIDContextKey = "passenger_id"

type PassengerAccessTokenParser interface {
	ParsePassengerAccessToken(token string) (passengerapp.TokenClaims, error)
}

type PassengerLookupRepository interface {
	GetByID(ctx context.Context, passengerID uuid.UUID) (domain.Passenger, error)
}

func AuthenticatePassengerAccessToken(parser PassengerAccessTokenParser, repository PassengerLookupRepository) gin.HandlerFunc {
	return func(context *gin.Context) {
		token := bearerToken(context.GetHeader("Authorization"))
		if token == "" {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token is missing", nil)
			context.Abort()
			return
		}

		claims, err := parser.ParsePassengerAccessToken(token)
		if err != nil {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token is invalid", nil)
			context.Abort()
			return
		}
		if claims.Subject == uuid.Nil || claims.Role != domain.UserRolePassenger {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token claims are invalid", nil)
			context.Abort()
			return
		}

		passengerRecord, err := repository.GetByID(context.Request.Context(), claims.Subject)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token is invalid", nil)
			} else {
				response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token is invalid", nil)
			}
			context.Abort()
			return
		}
		if !passengerRecord.IsActive {
			response.Fail(context, http.StatusForbidden, response.CodePassengerBlocked, "Passenger is blocked", nil)
			context.Abort()
			return
		}

		context.Set(PassengerIDContextKey, passengerRecord.ID)
		context.Set(UserRoleContextKey, domain.UserRolePassenger)
		context.Next()
	}
}

func PassengerIDFromContext(context *gin.Context) (uuid.UUID, bool) {
	value, exists := context.Get(PassengerIDContextKey)
	if !exists {
		return uuid.Nil, false
	}

	switch id := value.(type) {
	case uuid.UUID:
		return id, id != uuid.Nil
	default:
		return uuid.Nil, false
	}
}
