package engine

import (
	"testing"

	"github.com/cmalafis10/auth-accelerator/internal/env"
)

func TestSelect_RHBKCACOn420(t *testing.T) {
	e := env.Defaults() // rhbk + cac + 4.20
	p, err := Select(e)
	if err != nil {
		t.Fatalf("expected a match, got error: %v", err)
	}
	if p.ID != "rhbk-cac-external-oidc" {
		t.Fatalf("matched wrong pattern: %s", p.ID)
	}
}

func TestSelect_RejectsPre420(t *testing.T) {
	e := env.Defaults()
	e.OCPVersion = "4.19" // external OIDC not GA yet
	if _, err := Select(e); err == nil {
		t.Fatal("expected no match for OCP 4.19, got a pattern")
	}
}

func TestSelect_KeycloakAlias(t *testing.T) {
	e := env.Defaults()
	e.IDP = "keycloak"
	if _, err := Select(e); err != nil {
		t.Fatalf("keycloak should match the rhbk pattern: %v", err)
	}
}

func TestSelect_OktaCBAOn420(t *testing.T) {
	e := env.Defaults()
	e.IDP = "okta"
	e.SmartCard = "piv"
	p, err := Select(e)
	if err != nil {
		t.Fatalf("expected okta pattern to match, got error: %v", err)
	}
	if p.ID != "okta-cba-external-oidc" {
		t.Fatalf("matched wrong pattern: %s", p.ID)
	}
}

func TestSelect_OktaRejectsPre420(t *testing.T) {
	e := env.Defaults()
	e.IDP = "okta"
	e.SmartCard = "piv"
	e.OCPVersion = "4.19" // external OIDC not GA yet
	if _, err := Select(e); err == nil {
		t.Fatal("expected no match for okta on OCP 4.19, got a pattern")
	}
}

func TestSelect_OktaRejectsNonSmartcard(t *testing.T) {
	e := env.Defaults()
	e.IDP = "okta"
	e.SmartCard = "none" // CBA pattern requires cac|piv
	if _, err := Select(e); err == nil {
		t.Fatal("expected no match for okta without a smart card, got a pattern")
	}
}

func TestSelect_EntraCBAOn420(t *testing.T) {
	e := env.Defaults()
	e.IDP = "entra"
	e.SmartCard = "cac"
	p, err := Select(e)
	if err != nil {
		t.Fatalf("expected entra pattern to match, got error: %v", err)
	}
	if p.ID != "entra-cba-external-oidc" {
		t.Fatalf("matched wrong pattern: %s", p.ID)
	}
}

func TestSelect_EntraRejectsPre420(t *testing.T) {
	e := env.Defaults()
	e.IDP = "entra"
	e.SmartCard = "cac"
	e.OCPVersion = "4.19" // external OIDC not GA yet
	if _, err := Select(e); err == nil {
		t.Fatal("expected no match for entra on OCP 4.19, got a pattern")
	}
}
