package security

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

type NumericCodeGenerator struct{}

func NewNumericCodeGenerator() *NumericCodeGenerator {
	return &NumericCodeGenerator{}
}

func (generator *NumericCodeGenerator) GenerateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("code length must be positive")
	}

	code := make([]byte, length)
	for index := range code {
		value, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate numeric code: %w", err)
		}
		code[index] = byte('0' + value.Int64())
	}

	return string(code), nil
}

type BCryptCodeHasher struct {
	cost int
}

func NewBCryptCodeHasher(cost int) *BCryptCodeHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &BCryptCodeHasher{cost: cost}
}

func (hasher *BCryptCodeHasher) HashCode(code string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(code), hasher.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash verification code: %w", err)
	}

	return string(hash), nil
}

func (hasher *BCryptCodeHasher) CompareCodeAndHash(code string, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)); err != nil {
		return fmt.Errorf("bcrypt compare verification code: %w", err)
	}

	return nil
}
