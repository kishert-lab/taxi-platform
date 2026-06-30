package domain

import (
	"time"

	"github.com/google/uuid"
)

type CarClass struct {
	ID             uuid.UUID
	Code           string
	Name           string
	Description    string
	BasePrice      Money
	PricePerKM     Money
	PricePerMinute Money
	MinimumPrice   Money
	IsActive       bool
	SortOrder      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
