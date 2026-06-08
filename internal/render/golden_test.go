package render

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/cmalafis10/auth-accelerator/internal/engine"
	"github.com/cmalafis10/auth-accelerator/internal/env"
)

// update regenerates the golden files instead of comparing against them.
// Run: go test ./internal/render -run TestGolden -update
var update = flag.Bool("update", false, "update golden files")

// dateRe normalizes the rendered date (YYYY-MM-DD) so goldens stay stable.
var dateRe = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// goldenCases pins one representative Environment per IdP. These drive the
// snapshot of every generated artifact, so the exact YAML/runbook can't drift
// silently — any template change must be accepted with -update.
func goldenCases() []struct {
	id  string
	env env.Environment
} {
	mk := func(idp, card, pki, issuer string) env.Environment {
		e := env.Defaults()
		e.IDP, e.SmartCard, e.PKI, e.IssuerURL = idp, card, pki, issuer
		return e
	}
	return []struct {
		id  string
		env env.Environment
	}{
		{"rhbk", mk("rhbk", "cac", "dod", "https://sso.example.mil/realms/fed")},
		{"okta", mk("okta", "piv", "federal", "https://example.okta.com")},
		{"entra", mk("entra", "cac", "dod", "https://login.microsoftonline.com/TENANT/v2.0")},
		{"ping", mk("ping", "cac", "dod", "https://pingfed.example.mil")},
	}
}

func TestGolden(t *testing.T) {
	for _, c := range goldenCases() {
		t.Run(c.id, func(t *testing.T) {
			p, err := engine.Select(c.env)
			if err != nil {
				t.Fatalf("Select: %v", err)
			}
			e := c.env
			e.ApplyWiring(p.Wiring)

			dir := t.TempDir()
			written, err := Generate(p, e, dir)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			for _, w := range written {
				name := filepath.Base(w)
				raw, err := os.ReadFile(w)
				if err != nil {
					t.Fatalf("read %s: %v", w, err)
				}

				// The Authentication CR must be valid, parseable YAML.
				if name == "authentication-cr.yaml" {
					var doc map[string]any
					if err := yaml.Unmarshal(raw, &doc); err != nil {
						t.Errorf("%s/%s is not valid YAML: %v", c.id, name, err)
					}
				}

				got := dateRe.ReplaceAllString(string(raw), "<DATE>")
				goldenPath := filepath.Join("testdata", "golden", c.id, name)

				if *update {
					if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
					continue
				}

				want, err := os.ReadFile(goldenPath)
				if err != nil {
					t.Fatalf("missing golden %s — run: go test ./internal/render -run TestGolden -update\n%v", goldenPath, err)
				}
				if got != string(want) {
					t.Errorf("%s/%s drifted from golden (run -update to accept the change):\n--- got ---\n%s\n--- want ---\n%s",
						c.id, name, got, string(want))
				}
			}
		})
	}
}
