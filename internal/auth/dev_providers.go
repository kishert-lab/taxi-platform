package auth

import (
	"context"

	"go.uber.org/zap"
)

type LoggingSMSProvider struct {
	logger *zap.Logger
}

func NewLoggingSMSProvider(logger *zap.Logger) *LoggingSMSProvider {
	return &LoggingSMSProvider{logger: logger}
}

func (provider *LoggingSMSProvider) SendVerificationCode(_ context.Context, phone string, code string) error {
	provider.logger.Info("registration sms confirmation code issued", zap.String("phone", phone), zap.String("verification_code", code))
	return nil
}

type LoggingEmailProvider struct {
	logger *zap.Logger
}

func NewLoggingEmailProvider(logger *zap.Logger) *LoggingEmailProvider {
	return &LoggingEmailProvider{logger: logger}
}

func (provider *LoggingEmailProvider) SendEmailConfirmationCode(_ context.Context, email string, code string) error {
	provider.logger.Info("email confirmation code issued", zap.String("email", email), zap.String("verification_code", code))
	return nil
}
