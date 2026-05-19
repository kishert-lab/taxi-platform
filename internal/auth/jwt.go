package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	issuer        string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

type TokenManagerConfig struct {
	AccessSecret  string
	RefreshSecret string
	Issuer        string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type TokenClaims struct {
	Subject   uuid.UUID       `json:"sub"`
	Role      domain.UserRole `json:"role"`
	TokenType string          `json:"typ"`
	TokenID   uuid.UUID       `json:"jti"`
	Issuer    string          `json:"iss"`
	IssuedAt  int64           `json:"iat"`
	ExpiresAt int64           `json:"exp"`
}

func NewTokenManager(config TokenManagerConfig) *TokenManager {
	return &TokenManager{
		accessSecret:  []byte(config.AccessSecret),
		refreshSecret: []byte(config.RefreshSecret),
		issuer:        config.Issuer,
		accessTTL:     config.AccessTTL,
		refreshTTL:    config.RefreshTTL,
	}
}

func (manager *TokenManager) IssueTokenPair(userID uuid.UUID, role domain.UserRole, now time.Time) (accessToken string, refreshToken string, refreshExpiresAt time.Time, err error) {
	accessToken, err = manager.issueToken(userID, role, TokenTypeAccess, manager.accessTTL, now)
	if err != nil {
		return "", "", time.Time{}, err
	}

	refreshExpiresAt = now.Add(manager.refreshTTL)
	refreshToken, err = manager.issueToken(userID, role, TokenTypeRefresh, manager.refreshTTL, now)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return accessToken, refreshToken, refreshExpiresAt, nil
}

func (manager *TokenManager) ParseAccessToken(token string) (TokenClaims, error) {
	return manager.parseToken(token, TokenTypeAccess)
}

func (manager *TokenManager) ParseRefreshToken(token string) (TokenClaims, error) {
	return manager.parseToken(token, TokenTypeRefresh)
}

func (manager *TokenManager) AccessTTLSeconds() int64 {
	return int64(manager.accessTTL.Seconds())
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (manager *TokenManager) issueToken(userID uuid.UUID, role domain.UserRole, tokenType string, ttl time.Duration, now time.Time) (string, error) {
	claims := TokenClaims{
		Subject:   userID,
		Role:      role,
		TokenType: tokenType,
		TokenID:   uuid.New(),
		Issuer:    manager.issuer,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}

	headerJSON, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsignedToken := encodedHeader + "." + encodedClaims
	signature := manager.sign(unsignedToken, tokenType)

	return unsignedToken + "." + signature, nil
}

func (manager *TokenManager) parseToken(token string, expectedType string) (TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, ErrInvalidToken
	}

	unsignedToken := parts[0] + "." + parts[1]
	expectedSignature := manager.sign(unsignedToken, expectedType)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return TokenClaims{}, ErrInvalidToken
	}

	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return TokenClaims{}, ErrInvalidToken
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return TokenClaims{}, ErrInvalidToken
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return TokenClaims{}, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return TokenClaims{}, ErrInvalidToken
	}
	var claims TokenClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return TokenClaims{}, ErrInvalidToken
	}
	if claims.TokenType != expectedType || claims.Issuer != manager.issuer || time.Now().UTC().Unix() >= claims.ExpiresAt {
		return TokenClaims{}, ErrInvalidToken
	}

	return claims, nil
}

func (manager *TokenManager) sign(unsignedToken string, tokenType string) string {
	secret := manager.accessSecret
	if tokenType == TokenTypeRefresh {
		secret = manager.refreshSecret
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsignedToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
