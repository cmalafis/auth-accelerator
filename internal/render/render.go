// Package render fills the embedded templates from the chosen pattern + env and
// writes the artifacts (Authentication CR, IdP recipe, runbook, ADR) to a dir.
package render

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/cmalafis10/auth-accelerator/internal/catalog"
	"github.com/cmalafis10/auth-accelerator/internal/env"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// view is the data handed to every template.
type view struct {
	Env     env.Environment
	Pattern catalog.AuthPattern
	Date    string
}

// Generate renders all of the pattern's manifests into outDir.
func Generate(p *catalog.AuthPattern, e env.Environment, outDir string) ([]string, error) {
	tmpl, err := template.New("auth-accelerator").
		Funcs(template.FuncMap{"add1": func(i int) int { return i + 1 }}).
		ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	v := view{Env: e, Pattern: *p, Date: time.Now().Format("2006-01-02")}
	written := make([]string, 0, len(p.Manifests))

	for _, m := range p.Manifests {
		outPath := filepath.Join(outDir, m.Out)
		f, err := os.Create(outPath)
		if err != nil {
			return written, fmt.Errorf("creating %s: %w", outPath, err)
		}
		if err := tmpl.ExecuteTemplate(f, m.Template, v); err != nil {
			f.Close()
			return written, fmt.Errorf("rendering %s: %w", m.Template, err)
		}
		f.Close()
		written = append(written, outPath)
	}
	return written, nil
}
