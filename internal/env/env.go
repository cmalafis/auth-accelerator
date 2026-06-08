// Package env models the admin's available components — the answers that drive
// the decision engine. The interactive Prompt here is intentionally minimal
// (stdlib only); replace it with charmbracelet/huh in the first Claude Code session.
package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Environment is everything the tool needs to pick a topology and render config.
type Environment struct {
	OCPVersion  string // e.g. "4.20"
	OCPFlavor   string // self-managed | rosa | aro | okd
	IDP         string // rhbk | keycloak | entra | okta | ping | adfs | none
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

// Validate checks the required fields are present and sane.
func (e Environment) Validate() error {
	if e.OCPVersion == "" {
		return fmt.Errorf("OCPVersion is required (e.g. 4.20)")
	}
	if e.IDP == "" {
		return fmt.Errorf("IDP is required (e.g. rhbk)")
	}
	if e.IssuerURL == "" {
		return fmt.Errorf("IssuerURL is required")
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
	e.IDP = ask("Identity provider (rhbk|keycloak|entra|okta|ping|adfs)", e.IDP)
	e.SmartCard = ask("Smart card (cac|piv|eca)", e.SmartCard)
	e.PKI = ask("PKI (dod|federal|eca)", e.PKI)
	e.GroupSource = ask("Group source (claims|ldap)", e.GroupSource)
	e.IssuerURL = ask("OIDC issuer URL", e.IssuerURL)
	return e
}
