package push

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type googleServicesConfig struct {
	ProjectInfo struct {
		ProjectID string `json:"project_id"`
	} `json:"project_info"`
}

func ProjectIDFromGoogleServicesFile(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read google services file: %w", err)
	}

	var config googleServicesConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return "", fmt.Errorf("parse google services file: %w", err)
	}

	projectID := strings.TrimSpace(config.ProjectInfo.ProjectID)
	if projectID == "" {
		return "", fmt.Errorf("google services file project_id is empty")
	}

	return projectID, nil
}
