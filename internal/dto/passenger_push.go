package dto

type PassengerPushTokenRequest struct {
	Token    string `json:"token" binding:"required" example:"fcm_device_token"`
	Platform string `json:"platform" binding:"required,oneof=android ios web" example:"android"`
	DeviceID string `json:"device_id,omitempty" example:"pixel-8-pro"`
}

type PassengerPushTokenResponse struct {
	Token    string `json:"token" example:"fcm_device_token"`
	Platform string `json:"platform" example:"android"`
	DeviceID string `json:"device_id,omitempty" example:"pixel-8-pro"`
}
