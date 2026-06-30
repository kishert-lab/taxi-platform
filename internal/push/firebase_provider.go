package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const firebaseMessagingScope = "https://www.googleapis.com/auth/firebase.messaging"

type FirebaseProvider struct {
	projectID   string
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
}

func NewFirebaseProvider(projectID string, credentialsFile string) (*FirebaseProvider, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, fmt.Errorf("firebase project id is required")
	}
	if strings.TrimSpace(credentialsFile) == "" {
		return nil, fmt.Errorf("firebase credentials file is required")
	}

	credentialsBytes, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read firebase credentials file: %w", err)
	}
	config, err := google.JWTConfigFromJSON(credentialsBytes, firebaseMessagingScope)
	if err != nil {
		return nil, fmt.Errorf("parse firebase credentials: %w", err)
	}

	return &FirebaseProvider{
		projectID:   projectID,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		tokenSource: config.TokenSource(context.Background()),
	}, nil
}

func (provider *FirebaseProvider) SendToTokens(ctx context.Context, tokens []string, notification Notification) error {
	if len(tokens) == 0 {
		return nil
	}
	token, err := provider.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("get firebase oauth token: %w", err)
	}

	for _, deviceToken := range tokens {
		if err := provider.sendToToken(ctx, token.AccessToken, deviceToken, notification); err != nil {
			return err
		}
	}
	return nil
}

func (provider *FirebaseProvider) sendToToken(ctx context.Context, accessToken string, deviceToken string, notification Notification) error {
	requestBody := map[string]any{
		"message": map[string]any{
			"token": deviceToken,
			"notification": map[string]string{
				"title": notification.Title,
				"body":  notification.Body,
			},
			"data": notification.Data,
			"android": map[string]string{
				"priority": "high",
			},
		},
	}

	bytesBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal firebase message: %w", err)
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", provider.projectID),
		bytes.NewReader(bytesBody),
	)
	if err != nil {
		return fmt.Errorf("build firebase request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := provider.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send firebase message: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("firebase send failed: status=%d body=%s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
