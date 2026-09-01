package asset

import (
	"testing"
)

// TestLookup covers case-insensitive resolution of a verified code and that
// an unrecognised code is reported as absent rather than guessed at.
func TestLookup(t *testing.T) {
	cases := []struct {
		code string
		want Asset
		ok   bool
	}{
		{"USDC", USDC(), true},
		{"usdc", USDC(), true},
		{" NgNc ", NGNC(), true},
		{"NOTREAL", Asset{}, false},
		{"", Asset{}, false},
	}
	for _, c := range cases {
		got, ok := Lookup(c.code)
		if ok != c.ok {
			t.Errorf("Lookup(%q) ok = %v, want %v", c.code, ok, c.ok)
			continue
		}
		if ok && !got.Equal(c.want) {
			t.Errorf("Lookup(%q) = %+v, want %+v", c.code, got, c.want)
		}
	}
}

// TestKnownCodes pins that the list is sorted, since callers render it
// directly (e.g. in CLI help output) and an unsorted map iteration order
// would make that output nondeterministic.
func TestKnownCodes(t *testing.T) {
	got := KnownCodes()
	want := []string{"EURMTL", "GHSC", "KESC", "NGNC", "NGNT", "PYUSD", "USDC", "USDZ", "ZARZ"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("KnownCodes() = %v, want %v", got, want)
	}
}

// TestFiatPeg covers a registered token, a correct code from the wrong
// issuer, and native XLM — the three cases the doc comment on FiatPeg
// promises not to guess on.
func TestFiatPeg(t *testing.T) {
	if peg, ok := FiatPeg(NGNC()); !ok || peg != "NGN" {
		t.Errorf("FiatPeg(NGNC()) = (%q, %v), want (\"NGN\", true)", peg, ok)
	}
	if peg, ok := FiatPeg(GHSC()); !ok || peg != "GHS" {
		t.Errorf("FiatPeg(GHSC()) = (%q, %v), want (\"GHS\", true)", peg, ok)
	}
	if peg, ok := FiatPeg(KESC()); !ok || peg != "KES" {
		t.Errorf("FiatPeg(KESC()) = (%q, %v), want (\"KES\", true)", peg, ok)
	}

	// The expanded registry: every entry verified 2026-08-26 from the
	// issuer's own stellar.toml.
	for _, c := range []struct {
		a   Asset
		peg string
	}{
		{NGNT(), "NGN"},
		{USDZ(), "USD"},
		{ZARZ(), "ZAR"},
		{EURMTL(), "EUR"},
		{PYUSD(), "USD"},
	} {
		if peg, ok := FiatPeg(c.a); !ok || peg != c.peg {
			t.Errorf("FiatPeg(%s) = (%q, %v), want (%q, true)", c.a, peg, ok, c.peg)
		}
	}

	impostor := Stellar("NGNC", "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF5")
	if _, ok := FiatPeg(impostor); ok {
		t.Error("FiatPeg must return false for the right code from the wrong issuer")
	}

	if _, ok := FiatPeg(Native()); ok {
		t.Error("FiatPeg must return false for native XLM, a bridge asset")
	}

	if _, ok := FiatPeg(USDC()); ok {
		t.Error("FiatPeg must return false for a verified token with no registered peg")
	}
}

// TestClassifyHop pins the three-way hop classification that route.classify
// relies on: fiat-pegged tokens are dependencies, native XLM and registered
// non-fiat tokens are bridges, and everything else is unknown — including an
// unregistered token whose code matches a registered fiat token.
func TestClassifyHop(t *testing.T) {
	cases := []struct {
		name string
		a    Asset
		want HopKind
	}{
		{"NGNC is a fiat dependency", NGNC(), HopFiat},
		{"the expanded registry is fiat", USDZ(), HopFiat},
		{"PYUSD is fiat", PYUSD(), HopFiat},
		{"native XLM is a bridge", Native(), HopBridge},
		{"USDC is a registered non-fiat bridge", USDC(), HopBridge},
		{"an unregistered token is unknown", Stellar("BLND", "GBLNDISS1234567890123456789012345678901234567890123456789"), HopUnknown},
		{"an impostor with a registered code is unknown", Stellar("NGNC", "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF5"), HopUnknown},
		{"off-chain fiat is unknown as a hop", NGN(), HopUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyHop(tc.a); got != tc.want {
				t.Errorf("ClassifyHop(%s) = %v, want %v", tc.a, got, tc.want)
			}
		})
	}
}

// TestClassifyHopString keeps the human forms stable for readers.
func TestClassifyHopString(t *testing.T) {
	cases := map[HopKind]string{
		HopFiat:    "fiat",
		HopBridge:  "bridge",
		HopUnknown: "unknown",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("HopKind(%d).String() = %q, want %q", k, got, want)
		}
	}
}

// TestIsFiatToken exercises the same cases through the boolean-only helper,
// since callers on the hot classification path use this form directly.
func TestIsFiatToken(t *testing.T) {
	for _, a := range []Asset{NGNC(), GHSC(), KESC(), NGNT()} {
		if !IsFiatToken(a) {
			t.Errorf("IsFiatToken(%s) = false, want true", a)
		}
	}

	impostor := Stellar("NGNC", "GAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAWHF5")
	if IsFiatToken(impostor) {
		t.Error("IsFiatToken must return false for an unregistered issuer")
	}
	if IsFiatToken(Native()) {
		t.Error("IsFiatToken must return false for XLM, a bridge asset")
	}
}

// TestRegistryCompleteness ensures that every registered entry is valid,
// and that every corridor destination token has a non-empty fiat peg, SEP-1
// status, verification date, source URL, and home domain.
func TestRegistryCompleteness(t *testing.T) {
	entries := Registry()
	if len(entries) == 0 {
		t.Fatal("Registry() returned no entries")
	}

	for _, e := range entries {
		if err := ValidateEntry(e); err != nil {
			t.Errorf("ValidateEntry(%+v) failed: %v", e, err)
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
