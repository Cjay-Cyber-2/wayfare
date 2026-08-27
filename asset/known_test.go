package asset

import (
	"testing"
)

func TestRegistryVerificationDates(t *testing.T) {
	for _, entry := range Registry() {
		if entry.VerificationDate == "" {
			t.Errorf("asset %s lacks a verification date", entry.Code)
		}
		if entry.SourceURL == "" {
			t.Errorf("asset %s lacks a source URL", entry.Code)
		}
		if entry.HomeDomain == "" {
			t.Errorf("asset %s lacks a home domain", entry.Code)
		}
		if entry.Status == "" {
			t.Errorf("asset %s lacks a status", entry.Code)
		}
	}
}

func TestValidateEntryRequiresVerificationDate(t *testing.T) {
	err := ValidateEntry(Entry{
		Code:             "TEST",
		Issuer:           LinkIOIssuer,
		Peg:              "TST",
		Status:           "live",
		VerificationDate: "",
		SourceURL:        "https://example.com/.well-known/stellar.toml",
		HomeDomain:       "example.com",
	})
	if err == nil {
		പരമായ("expected error for missing verification date, got nil")
	}
}
