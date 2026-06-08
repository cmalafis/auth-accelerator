// Package env models the admin's available components — the answers that drive
// the decision engine. The interactive Prompt here is intentionally minimal
// (stdlib only); replace it with charmbracelet/huh in the first Claude Code session.
package env

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Environment is everything the tool needs to pick a topology and render config.
type Environment struct {
	OCPVersion  string // e.g. "4.20"
	OCPFlavor   string // self-managed | rosa | aro | okd
	IDP         string // rhbk | keycloak | okta | entra | ping
	SmartCard   string // cac | piv | eca
	PKI         string // dod | federal | eca
	FIPS        bool
	AirGapped   bool
	GroupSource string // claims | ldap

	// OIDC wiring rendered into the Authentication CR.
	ProviderName      string
	IssuerURL         string
	ConsoleClientID   string
	CLIClientID       string
	ConsoleSecretName string // console client secret in openshift-config
	CABundleConfigMap string
	UsernameClaim     string
	GroupsClaim       string
}

// OIDCDefaults is the per-IdP OIDC wiring an AuthPattern carries. The CLI applies
// the matched pattern's defaults to the Environment so each IdP renders its own
// provider name, claim names, and client wiring (see Environment.ApplyWiring).
// IssuerURL is deliberately absent: it is deployment-specific and comes from --issuer.
type OIDCDefaults struct {
	ProviderName      string
	ConsoleClientID   string
	CLIClientID       string
	ConsoleSecretName string
	UsernameClaim     string
	GroupsClaim       string
	CABundleConfigMap string
}

// ApplyWiring copies the non-empty fields of d onto the Environment. Non-empty
// wins, so a pattern may leave a field blank to keep the existing default.
func (e *Environment) ApplyWiring(d OIDCDefaults) {
	if d.ProviderName != "" {
		e.ProviderName = d.ProviderName
	}
	if d.ConsoleClientID != "" {
		e.ConsoleClientID = d.ConsoleClientID
	}
	if d.CLIClientID != "" {
		e.CLIClientID = d.CLIClientID
	}
	if d.ConsoleSecretName != "" {
		e.ConsoleSecretName = d.ConsoleSecretName
	}
	if d.UsernameClaim != "" {
		e.UsernameClaim = d.UsernameClaim
	}
	if d.GroupsClaim != "" {
		e.GroupsClaim = d.GroupsClaim
	}
	if d.CABundleConfigMap != "" {
		e.CABundleConfigMap = d.CABundleConfigMap
	}
}

// Defaults returns an Environment seeded with sensible federal defaults.
func Defaults() Environment {
	return Environment{
		OCPVersion:        "4.20",
		OCPFlavor:         "self-managed",
		IDP:               "rhbk",
		SmartCard:         "cac",
		PKI:               "dod",
		GroupSource:       "claims",
		ProviderName:      "rhbk-cac",
		IssuerURL:         "https://sso.example.mil/realms/fed",
		ConsoleClientID:   "openshift-console",
		CLIClientID:       "openshift-cli",
		ConsoleSecretName: "console-oidc-secret",
		// CABundleConfigMap is intentionally unset here: it is supplied per-IdP
		// via AuthPattern.Wiring. Self-hosted IdPs (RHBK) set it; IdPs behind a
		// public CA (Entra) leave it empty so the CR omits issuerCertificateAuthority.
		UsernameClaim: "preferred_username",
		GroupsClaim:   "groups",
	}
}

// Allowed values for the enum fields, in the order shown to users. These are
// the syntactically valid inputs; the engine/catalog decides which combinations
// actually resolve to a pattern.
var (
	allowedIDP       = []string{"rhbk", "keycloak", "okta", "entra", "ping"}
	allowedSmartCard = []string{"cac", "piv", "eca"}
	allowedPKI       = []string{"dod", "federal", "eca"}
	allowedFlavor    = []string{"self-managed", "rosa", "aro", "okd"}
	allowedGroups    = []string{"claims", "ldap"}
)

// AllowedIDPs returns the identity-provider keys the tool accepts (a copy, so
// callers can't mutate the canonical list). Used for user-facing messages.
func AllowedIDPs() []string { return append([]string(nil), allowedIDP...) }

// oneOf returns a friendly error if val is not in allowed.
func oneOf(flag, val string, allowed []string) error {
	for _, a := range allowed {
		if val == a {
			return nil
		}
	}
	return fmt.Errorf("unknown %s %q; choose one of: %s", flag, val, strings.Join(allowed, ", "))
}

// Validate checks the required fields are present and that every value is one
// the tool understands, with messages aimed at a non-developer admin.
func (e Environment) Validate() error {
	if e.OCPVersion == "" {
		return fmt.Errorf("OpenShift version is required (--ocp, e.g. 4.20)")
	}
	if e.IDP == "" {
		return fmt.Errorf("identity provider is required (--idp, e.g. rhbk)")
	}
	if err := oneOf("--idp", e.IDP, allowedIDP); err != nil {
		return err
	}
	if err := oneOf("--smartcard", e.SmartCard, allowedSmartCard); err != nil {
		return err
	}
	if err := oneOf("--pki", e.PKI, allowedPKI); err != nil {
		return err
	}
	if err := oneOf("--flavor", e.OCPFlavor, allowedFlavor); err != nil {
		return err
	}
	if err := oneOf("--groups", e.GroupSource, allowedGroups); err != nil {
		return err
	}
	if e.IssuerURL == "" {
		return fmt.Errorf("OIDC issuer URL is required (--issuer)")
	}
	if err := validateIssuerURL(e.IssuerURL); err != nil {
		return err
	}
	return nil
}

// validateIssuerURL requires a well-formed https URL (OIDC issuers are https).
func validateIssuerURL(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("invalid issuer URL %q: %v", s, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("issuer URL must start with https:// (got %q)", s)
	}
	if u.Host == "" {
		return fmt.Errorf("issuer URL %q is missing a host (expected e.g. https://idp.example.mil)", s)
	}
	return nil
}

// Prompt fills an Environment interactively from stdin. Minimal by design.
func Prompt(in *os.File) Environment {
	e := Defaults()
	r := bufio.NewReader(in)
	ask := func(label, cur string) string {
		fmt.Printf("%s [%s]: ", label, cur)
		line, _ := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return cur
		}
		return line
	}
	e.OCPVersion = ask("OpenShift version", e.OCPVersion)
	e.OCPFlavor = ask("OCP flavor (self-managed|rosa|aro|okd)", e.OCPFlavor)
	e.IDP = ask("Identity provider (rhbk|keycloak|okta|entra|ping)", e.IDP)
	e.SmartCard = ask("Smart card (cac|piv|eca)", e.SmartCard)
	e.PKI = ask("PKI (dod|federal|eca)", e.PKI)
	e.GroupSource = ask("Group source (claims|ldap)", e.GroupSource)
	e.IssuerURL = ask("OIDC issuer URL", e.IssuerURL)
	return e
}
