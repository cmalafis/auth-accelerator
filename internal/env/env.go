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
	CABundleConfigMap string
	UsernameClaim     string
	GroupsClaim       string
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
		CABundleConfigMap: "oidc-ca-bundle",
		UsernameClaim:     "preferred_username",
		GroupsClaim:       "groups",
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
