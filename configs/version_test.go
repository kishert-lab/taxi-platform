package configs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeApplicationVersionUsesLatestMigrationAsMajor(t *testing.T) {
	migrationsDirectory := t.TempDir()
	writeVersionedMigration(t, migrationsDirectory, "000026_transport_request_logs.up.sql")
	writeVersionedMigration(t, migrationsDirectory, "000027_privacy_policy_152fz_update.up.sql")

	version := normalizeApplicationVersion("0.1.3", migrationsDirectory)
	if version != "27.1.3" {
		t.Fatalf("expected 27.1.3, got %s", version)
	}
}

func TestNormalizeApplicationVersionFallsBackWithoutMigrations(t *testing.T) {
	version := normalizeApplicationVersion("1.2.3", filepath.Join(t.TempDir(), "missing"))
	if version != "1.2.3" {
		t.Fatalf("expected configured version fallback, got %s", version)
	}
}

func TestNormalizeApplicationVersionDefaultsWhenConfiguredVersionInvalid(t *testing.T) {
	migrationsDirectory := t.TempDir()
	writeVersionedMigration(t, migrationsDirectory, "000027_privacy_policy_152fz_update.up.sql")

	version := normalizeApplicationVersion("dev", migrationsDirectory)
	if version != "27.0.0" {
		t.Fatalf("expected 27.0.0, got %s", version)
	}
}

func writeVersionedMigration(t *testing.T, directory string, fileName string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, fileName), []byte("-- migration"), 0o644); err != nil {
		t.Fatalf("write migration %s: %v", fileName, err)
	}
}
