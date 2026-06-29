package domain

import "testing"

func TestNormalizePhone(t *testing.T) {
	t.Run("keeps international format", func(t *testing.T) {
		normalized, err := NormalizePhone("+79991234567")
		if err != nil {
			t.Fatalf("NormalizePhone returned error: %v", err)
		}
		if normalized != "+79991234567" {
			t.Fatalf("unexpected normalized phone: %s", normalized)
		}
	})

	t.Run("converts russian local prefix", func(t *testing.T) {
		normalized, err := NormalizePhone("8 (999) 123-45-67")
		if err != nil {
			t.Fatalf("NormalizePhone returned error: %v", err)
		}
		if normalized != "+79991234567" {
			t.Fatalf("unexpected normalized phone: %s", normalized)
		}
	})

	t.Run("rejects invalid phone", func(t *testing.T) {
		if _, err := NormalizePhone("123"); err == nil {
			t.Fatal("expected invalid phone error")
		}
	})
}
