// Package engine is the deterministic decision core: given an Environment it
// selects the right AuthPattern from the catalog. No I/O, no LLM — pure and
// unit-testable, exactly like the rules-engine pattern from earlier projects.
package engine

import (
	"fmt"

	"github.com/cmalafis10/auth-accelerator/internal/catalog"
	"github.com/cmalafis10/auth-accelerator/internal/env"
)

// Select returns the first catalog pattern whose Applies(env) is true.
func Select(e env.Environment) (*catalog.AuthPattern, error) {
	for i := range catalog.Patterns {
		if catalog.Patterns[i].Applies(e) {
			return &catalog.Patterns[i], nil
		}
	}
	return nil, fmt.Errorf("no auth pattern matches this environment (idp=%s, smartcard=%s, ocp=%s)",
		e.IDP, e.SmartCard, e.OCPVersion)
}
