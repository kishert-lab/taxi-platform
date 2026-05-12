package domain

import (
	"errors"
	"testing"
)

func TestValidateRequiredRegistrationConsent(t *testing.T) {
	t.Parallel()

	err := ValidateRequiredRegistrationConsent(true, true, "1.0", "1.0")
	if err != nil {
		t.Fatalf("expected consent to be valid: %v", err)
	}
}

func TestValidateRequiredRegistrationConsentRejectsMissingConsent(t *testing.T) {
	t.Parallel()

	err := ValidateRequiredRegistrationConsent(false, true, "1.0", "1.0")
	if !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("expected consent required error, got %v", err)
	}
}

func TestValidateRequiredRegistrationConsentRejectsMissingDocumentVersion(t *testing.T) {
	t.Parallel()

	err := ValidateRequiredRegistrationConsent(true, true, "", "1.0")
	if !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("expected consent required error, got %v", err)
	}
}
