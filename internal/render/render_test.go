package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cmalafis10/auth-accelerator/internal/engine"
	"github.com/cmalafis10/auth-accelerator/internal/env"
)

// renderFor mirrors the CLI flow: select the pattern for e, apply its wiring,
// render into a temp dir, and return the dir plus the written file basenames.
func renderFor(t *testing.T, e env.Environment) (string, map[string]string) {
	t.Helper()
	p, err := engine.Select(e)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	e.ApplyWiring(p.Wiring)

	dir := t.TempDir()
	written, err := Generate(p, e, dir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	files := make(map[string]string, len(written))
	for _, w := range written {
		b, err := os.ReadFile(w)
		if err != nil {
			t.Fatalf("read %s: %v", w, err)
		}
		files[filepath.Base(w)] = string(b)
	}
	return dir, files
}

func TestGenerate_OktaCBA(t *testing.T) {
	e := env.Defaults()
	e.IDP = "okta"
	e.SmartCard = "piv"
	e.PKI = "federal"
	e.IssuerURL = "https://example.okta.com"

	_, files := renderFor(t, e)

	for _, want := range []string{
		"authentication-cr.yaml",
		"okta-cba-recipe.md",
		"runbook.md",
		"adr-0001-smart-card-auth.md",
	} {
		if _, ok := files[want]; !ok {
			t.Errorf("expected output file %q, missing", want)
		}
	}

	cr := files["authentication-cr.yaml"]
	for _, want := range []string{
		"name: okta-cba",            // Okta provider name from Wiring
		"claim: email",              // Okta username claim from Wiring
		"oidcClients:",              // enhanced CR
		"componentName: console",    // console confidential client
		"name: console-oidc-secret", // clientSecret reference
		"https://example.okta.com",  // issuer from --issuer
	} {
		if !strings.Contains(cr, want) {
			t.Errorf("authentication-cr.yaml missing %q:\n%s", want, cr)
		}
	}

	if recipe := files["okta-cba-recipe.md"]; !strings.Contains(recipe, "Certificate-Based Authentication") {
		t.Errorf("okta-cba-recipe.md does not describe CBA:\n%s", recipe)
	}
}

func TestGenerate_RHBKStillHasOIDCClients(t *testing.T) {
	e := env.Defaults() // rhbk + cac + 4.20
	_, files := renderFor(t, e)

	cr := files["authentication-cr.yaml"]
	if !strings.Contains(cr, "oidcClients:") {
		t.Errorf("rhbk authentication-cr.yaml missing oidcClients block:\n%s", cr)
	}
	if !strings.Contains(cr, "name: rhbk-cac") {
		t.Errorf("rhbk provider name changed unexpectedly:\n%s", cr)
	}
	if !strings.Contains(cr, "claim: preferred_username") {
		t.Errorf("rhbk username claim changed unexpectedly:\n%s", cr)
	}
}
