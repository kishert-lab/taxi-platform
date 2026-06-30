package passenger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type LoggingSMSService struct {
	logger *zap.Logger
}

func NewLoggingSMSService(logger *zap.Logger) *LoggingSMSService {
	return &LoggingSMSService{logger: logger}
}

func (service *LoggingSMSService) SendCode(_ context.Context, phone string, code string) error {
	if service.logger != nil {
		service.logger.Info("passenger auth code fallback", zap.String("phone", phone), zap.String("code", code))
	}
	fmt.Printf("PASSENGER_AUTH_CODE phone=%s code=%s\n", phone, code)
	return nil
}
