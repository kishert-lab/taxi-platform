package dto

import "github.com/google/uuid"

type PassengerAuthRequestCodeRequest struct {
	Phone string `json:"phone" binding:"required" example:"+79124966126"`
}

type PassengerAuthRequestCodeResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Код подтверждения отправлен"`
}

type PassengerAuthConfirmCodeRequest struct {
	Phone string `json:"phone" binding:"required" example:"+79124966126"`
	Code  string `json:"code" binding:"required,min=4,max=6" example:"1234"`
	Name  string `json:"name,omitempty" binding:"omitempty,max=255" example:"Сергей"`
}

type PassengerMeResponse struct {
	ID        uuid.UUID `json:"id" example:"11111111-1111-1111-1111-111111111111"`
	Phone     string    `json:"phone" example:"+79124966126"`
	Name      string    `json:"name,omitempty" example:"Сергей"`
	Email     string    `json:"email,omitempty" example:"test@example.com"`
	AvatarURL string    `json:"avatar_url,omitempty" example:"https://cdn.example.com/passengers/111/avatar.jpg"`
}

type PassengerAuthTokenResponse struct {
	AccessToken  string              `json:"access_token" example:"eyJhbGciOi..."`
	RefreshToken string              `json:"refresh_token" example:"eyJhbGciOi..."`
	TokenType    string              `json:"token_type" example:"Bearer"`
	ExpiresIn    int64               `json:"expires_in" example:"900"`
	Passenger    PassengerMeResponse `json:"passenger"`
}

type PassengerAuthRefreshResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOi..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOi..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
}

type PassengerMePatchRequest struct {
	Name      *string `json:"name,omitempty" binding:"omitempty,max=255" example:"Сергей"`
	Email     *string `json:"email,omitempty" binding:"omitempty,email" example:"test@example.com"`
	AvatarURL *string `json:"avatar_url,omitempty" binding:"omitempty,url" example:"https://cdn.example.com/passengers/111/avatar.jpg"`
}
