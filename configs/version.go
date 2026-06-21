package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var migrationFilePattern = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

func normalizeApplicationVersion(configuredVersion string, migrationsDirectory string) string {
	majorVersion, err := latestMigrationVersion(migrationsDirectory)
	if err != nil || majorVersion <= 0 {
		return normalizeVersionFallback(configuredVersion)
	}

	minorVersion, patchVersion := parseMinorAndPatch(configuredVersion)
	return fmt.Sprintf("%d.%d.%d", majorVersion, minorVersion, patchVersion)
}

func latestMigrationVersion(migrationsDirectory string) (int, error) {
	entries, err := os.ReadDir(migrationsDirectory)
	if err != nil {
		return 0, fmt.Errorf("read migrations directory %s: %w", migrationsDirectory, err)
	}

	latestVersion := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := migrationFilePattern.FindStringSubmatch(filepath.Base(entry.Name()))
		if len(matches) != 2 {
			continue
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("parse migration version from %s: %w", entry.Name(), err)
		}
		if version > latestVersion {
			latestVersion = version
		}
	}

	if latestVersion == 0 {
		return 0, fmt.Errorf("no migration files found in %s", migrationsDirectory)
	}

	return latestVersion, nil
}

func parseMinorAndPatch(configuredVersion string) (int, int) {
	normalizedVersion := normalizeVersionFallback(configuredVersion)
	parts := strings.Split(normalizedVersion, ".")
	if len(parts) != 3 {
		return 0, 0
	}

	minorVersion, err := strconv.Atoi(parts[1])
	if err != nil {
		minorVersion = 0
	}

	patchVersion, err := strconv.Atoi(parts[2])
	if err != nil {
		patchVersion = 0
	}

	return minorVersion, patchVersion
}

func normalizeVersionFallback(configuredVersion string) string {
	parts := strings.Split(strings.TrimSpace(configuredVersion), ".")
	if len(parts) != 3 {
		return "0.0.0"
	}

	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return "0.0.0"
		}
	}

	return strings.Join(parts, ".")
}
