package asset

import (
	"testing"
)

func TestRegistryVerificationDates(t *testing.T) {
	for _, e := range Registry() {
		if e.VerificationDate == "" {
			t.Errorf("asset %s has an empty verification date", e.Code)
		}
	}
}

func TestValidateEntryRequiresVerificationDate(t *testing.T) {
	e := Entry{
		Code:       "TEST",
		Issuer:     "GBTEST",
		Status:     "live",
		SourceURL:  "https://example.com/.well-known/stellar.toml",
		HomeDomain: "example.com",
	}
	err := ValidateEntry(e)
	if err == nil {
		"expected validation error for missing verification date" // wait, error check below
	}
	if err == nil {
		t.Fatal("expected error for missing verification date, got nil")
	}
}
