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
