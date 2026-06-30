package passenger

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
	TokenTypePassenger = "passenger"
	TokenUseAccess     = "access"
	TokenUseRefresh    = "refresh"
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
	TokenType string          `json:"token_type"`
	TokenUse  string          `json:"token_use"`
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

func (manager *TokenManager) GeneratePassengerAccessToken(passengerID uuid.UUID, now time.Time) (string, error) {
	return manager.issueToken(passengerID, TokenUseAccess, manager.accessTTL, now)
}

func (manager *TokenManager) GeneratePassengerRefreshToken(passengerID uuid.UUID, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(manager.refreshTTL)
	token, err := manager.issueToken(passengerID, TokenUseRefresh, manager.refreshTTL, now)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func (manager *TokenManager) GenerateTokenPair(passengerID uuid.UUID, now time.Time) (string, string, time.Time, error) {
	accessToken, err := manager.GeneratePassengerAccessToken(passengerID, now)
	if err != nil {
		return "", "", time.Time{}, err
	}
	refreshToken, refreshExpiresAt, err := manager.GeneratePassengerRefreshToken(passengerID, now)
	if err != nil {
		return "", "", time.Time{}, err
	}

	return accessToken, refreshToken, refreshExpiresAt, nil
}

func (manager *TokenManager) ParsePassengerAccessToken(token string) (TokenClaims, error) {
	return manager.parseToken(token, TokenUseAccess)
}

func (manager *TokenManager) ParsePassengerRefreshToken(token string) (TokenClaims, error) {
	return manager.parseToken(token, TokenUseRefresh)
}

func (manager *TokenManager) AccessTTLSeconds() int64 {
	return int64(manager.accessTTL.Seconds())
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (manager *TokenManager) issueToken(passengerID uuid.UUID, tokenUse string, ttl time.Duration, now time.Time) (string, error) {
	claims := TokenClaims{
		Subject:   passengerID,
		Role:      domain.UserRolePassenger,
		TokenType: TokenTypePassenger,
		TokenUse:  tokenUse,
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
	signature := manager.sign(unsignedToken, tokenUse)

	return unsignedToken + "." + signature, nil
}

func (manager *TokenManager) parseToken(token string, expectedUse string) (TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}

	unsignedToken := parts[0] + "." + parts[1]
	expectedSignature := manager.sign(unsignedToken, expectedUse)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSignature)) {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}

	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}

	var claims TokenClaims
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}
	if claims.Subject == uuid.Nil ||
		claims.Role != domain.UserRolePassenger ||
		claims.TokenType != TokenTypePassenger ||
		claims.TokenUse != expectedUse ||
		claims.Issuer != manager.issuer ||
		time.Now().UTC().Unix() >= claims.ExpiresAt {
		return TokenClaims{}, domain.ErrPassengerTokenInvalid
	}

	return claims, nil
}

func (manager *TokenManager) sign(unsignedToken string, tokenUse string) string {
	secret := manager.accessSecret
	if tokenUse == TokenUseRefresh {
		secret = manager.refreshSecret
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(unsignedToken))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
