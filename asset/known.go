package asset

import (
	"fmt"
	"sort"
	"strings"
)

// Verified mainnet issuers.
//
// Every issuer here was read from the issuer's own published stellar.toml
// (SEP-1) rather than copied from a block explorer or a blog post. Asset code
// alone is never sufficient identification: anyone can issue a token called
// "USDC", and a router that matched on code alone would happily quote a
// worthless lookalike. The verification date is recorded because anchors do
// rotate issuers.
//
// Verification status, 2026-08-08, read from
// https://ngnc.online/.well-known/stellar.toml:
//
//   - NGNC   VERIFIED, status="live". Issued by LINK.IO LTD., pegged 1:1 to
//     NGN, anchor_asset_type="fiat". NETWORK_PASSPHRASE = public mainnet.
//
//   - GHSC   VERIFIED as published, status="pending". Same issuing account as
//     NGNC. The anchor itself does not declare this asset in service.
//
//   - KESC   VERIFIED as published, status="pending". Same issuing account.
//     Note the entry sets anchor_asset="KESC", naming its own token rather
//     than the ISO-4217 code KES that SEP-1 intends. Read as KES.
//
//   - USDC   NOT YET VERIFIED against circle.com's stellar.toml. This is the
//     widely-published Circle issuer and Horizon accepted it for live
//     orderbook and path queries, which proves it is a real, actively traded
//     issuer — but not that it is Circle's. Confirm before any mainnet
//     execution path ships. See VerifyAgainstTOML in package anchor.
//
// The pending status on GHSC and KESC is a first-class finding, not a detail
// to route around. Per SEP-1 only "live" means in service, and the monitor
// reports an asset its own issuer has not launched as exactly that rather
// than pricing it as though it were tradeable.
const (
	// USDCIssuer is Circle's mainnet USDC issuing account.
	USDCIssuer = "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"

	// LinkIOIssuer issues NGNC, GHSC and KESC from one account. It is the
	// single point of failure behind this issuer's entire African-fiat set.
	LinkIOIssuer = "GASBV6W7GGED66MXEVC7YZHTWWYMSVYEY35USF2HJZBLABLYIFQGXZY6"

	// NGNCIssuer is retained as the original name for LinkIOIssuer.
	NGNCIssuer = LinkIOIssuer
)

// Entry represents a verified asset or corridor registration record.
//
// Structured metadata fields (Status, VerificationDate, SourceURL, HomeDomain)
// are stored directly as data so that tools and APIs can inspect them.
type Entry struct {
	Code             string // Asset code, e.g. "NGNC", "USDC"
	Issuer           string // Stellar issuing account ID
	Peg              string // ISO-4217 fiat currency tracked (e.g. "NGN"); empty for non-fiat settlement assets
	Status           string // SEP-1 status declared by issuer: "live", "pending", "unverified", etc.
	VerificationDate string // Date verified against issuer's stellar.toml (YYYY-MM-DD)
	SourceURL        string // URL where stellar.toml was read
	HomeDomain       string // Domain publishing stellar.toml
}

// CorridorEntry is an alias for Entry for backward and semantic compatibility.
type CorridorEntry = Entry

// ValidateEntry checks that a registration entry has all required fields.
// All registered assets require Code, Issuer, Status, VerificationDate, SourceURL,
// and HomeDomain. Corridor destination tokens additionally require Peg.
func ValidateEntry(e Entry) error {
	if strings.TrimSpace(e.Code) == "" {
		return fmt.Errorf("asset code is required")
	}
	if strings.TrimSpace(e.Issuer) == "" {
		return fmt.Errorf("asset %s: issuer is required", e.Code)
	}
	if strings.TrimSpace(e.Status) == "" {
		return fmt.Errorf("asset %s: SEP-1 status is required", e.Code)
	}
	if strings.TrimSpace(e.VerificationDate) == "" {
		return fmt.Errorf("asset %s: verification date is required", e.Code)
	}
	if strings.TrimSpace(e.SourceURL) == "" {
		return fmt.Errorf("asset %s: source URL is required", e.Code)
	}
	if strings.TrimSpace(e.HomeDomain) == "" {
		return fmt.Errorf("asset %s: home domain is required", e.Code)
	}
	// USDC is the settlement asset senders start from; all other registered assets
	// are corridor destination tokens whose peg is mandatory.
	if e.Code != "USDC" {
		if strings.TrimSpace(e.Peg) == "" {
			return fmt.Errorf("asset %s: fiat peg is required for corridor tokens", e.Code)
		}
	}
	return nil
}

// registry is the single source of truth for all verified assets and corridors.
// Maps like known, fiatPegs, and homeDomains are automatically derived from this slice.
var registry = []Entry{
	{
		Code:             "USDC",
		Issuer:           USDCIssuer,
		Peg:              "",
		Status:           "unverified",
		VerificationDate: "2026-08-08",
		SourceURL:        "https://www.circle.com/usdc/.well-known/stellar.toml",
		HomeDomain:       "circle.com",
	},
	{
		Code:             "NGNC",
		Issuer:           LinkIOIssuer,
		Peg:              "NGN",
		Status:           "live",
		VerificationDate: "2026-08-08",
		SourceURL:        "https://ngnc.online/.well-known/stellar.toml",
		HomeDomain:       "ngnc.online",
	},
	{
		Code:             "GHSC",
		Issuer:           LinkIOIssuer,
		Peg:              "GHS",
		Status:           "pending",
		VerificationDate: "2026-08-08",
		SourceURL:        "https://ngnc.online/.well-known/stellar.toml",
		HomeDomain:       "ngnc.online",
	},
	{
		Code:             "KESC",
		Issuer:           LinkIOIssuer,
		Peg:              "KES",
		Status:           "pending",
		VerificationDate: "2026-08-08",
		SourceURL:        "https://ngnc.online/.well-known/stellar.toml",
		HomeDomain:       "ngnc.online",
	},
}

var (
	known       = make(map[string]Asset)
	fiatPegs    = make(map[string]string)
	homeDomains = make(map[string]string)
	entries     = make(map[string]Entry)
)

func init() {
	for _, e := range registry {
		if err := ValidateEntry(e); err != nil {
			panic(fmt.Sprintf("asset: invalid registry entry %q: %v", e.Code, err))
		}
		a := Stellar(e.Code, e.Issuer)
		known[e.Code] = a
		if e.Peg != "" {
			fiatPegs[e.Code+":"+e.Issuer] = e.Peg
		}
		if e.HomeDomain != "" {
			homeDomains[e.Issuer] = e.HomeDomain
		}
		entries[e.Code+":"+e.Issuer] = e
	}
}

// USDC is the settlement asset senders start from.
func USDC() Asset { return Stellar("USDC", USDCIssuer) }

// NGNC is the naira-denominated token that terminates the on-chain leg.
// Declared live by its issuer.
func NGNC() Asset { return Stellar("NGNC", LinkIOIssuer) }

// GHSC is the Ghanaian cedi token from the same issuer as NGNC. Its issuer
// declares it status="pending" — not in service.
func GHSC() Asset { return Stellar("GHSC", LinkIOIssuer) }

// KESC is the Kenyan shilling token from the same issuer as NGNC. Its issuer
// declares it status="pending" — not in service.
func KESC() Asset { return Stellar("KESC", LinkIOIssuer) }

// Lookup resolves a verified token by its code.
//
// It returns false for anything not explicitly verified, so an unrecognised
// code is an error rather than a guess at an issuer.
func Lookup(code string) (Asset, bool) {
	a, ok := known[strings.ToUpper(strings.TrimSpace(code))]
	return a, ok
}

// LookupEntry returns the registration record for a given asset.
func LookupEntry(a Asset) (Entry, bool) {
	if a.Kind != KindStellar || a.Issuer == "" {
		return Entry{}, false
	}
	e, ok := entries[a.Code+":"+a.Issuer]
	return e, ok
}

// LookupEntryByCode returns the registration record for a given asset code.
func LookupEntryByCode(code string) (Entry, bool) {
	a, ok := Lookup(code)
	if !ok {
		return Entry{}, false
	}
	return LookupEntry(a)
}

// Registry returns a copy of all registered entries.
func Registry() []Entry {
	out := make([]Entry, len(registry))
	copy(out, registry)
	return out
}

// FiatPegs returns a copy of the mapping from asset code and issuer to fiat currency.
func FiatPegs() map[string]string {
	out := make(map[string]string, len(fiatPegs))
	for k, v := range fiatPegs {
		out[k] = v
	}
	return out
}

// HomeDomains returns a copy of the mapping from issuer account to home domain.
func HomeDomains() map[string]string {
	out := make(map[string]string, len(homeDomains))
	for k, v := range homeDomains {
		out[k] = v
	}
	return out
}

// IsKnown reports whether an asset is explicitly registered.
func IsKnown(a Asset) bool {
	if a.Kind != KindStellar {
		return false
	}
	_, ok := entries[a.Code+":"+a.Issuer]
	return ok
}

// FiatPeg returns the ISO currency code pegged by a registered asset, if any.
func FiatPeg(a Asset) bool {
	_, ok := fiatPegs[a.Code+":"+a.Issuer]
	return ok
}

// ListKnown returns a sorted slice of all registered asset codes.
func ListKnown() []string {
	out := make([]string, 0, len(known))
	for code := range known {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}
