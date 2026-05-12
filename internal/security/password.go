package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BCryptPasswordHasher hashes user passwords with bcrypt.
type BCryptPasswordHasher struct {
	cost int
}

func NewBCryptPasswordHasher(cost int) *BCryptPasswordHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &BCryptPasswordHasher{cost: cost}
}

func (hasher *BCryptPasswordHasher) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), hasher.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash password: %w", err)
	}

	return string(hash), nil
}

func (hasher *BCryptPasswordHasher) ComparePasswordAndHash(password string, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("bcrypt compare password: %w", err)
	}

	return nil
}
