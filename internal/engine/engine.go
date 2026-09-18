// Package engine is the deterministic decision core: given an Environment it
// selects the right AuthPattern from the catalog. No I/O, no LLM — pure and
// unit-testable, exactly like the rules-engine pattern from earlier projects.
package engine

import (
	"fmt"
	"strings"

	"github.com/cmalafis/auth-accelerator/internal/catalog"
	"github.com/cmalafis/auth-accelerator/internal/env"
)

// Select returns the first catalog pattern whose Applies(env) is true.
func Select(e env.Environment) (*catalog.AuthPattern, error) {
	for i := range catalog.Patterns {
		if catalog.Patterns[i].Applies(e) {
			return &catalog.Patterns[i], nil
		}
	}
	return nil, noMatchError(e)
}

// noMatchError explains, in priority order, the most likely reason no pattern
// matched — so a non-developer admin knows what to change rather than seeing a
// bare "no match".
func noMatchError(e env.Environment) error {
	if e.SmartCard != "cac" && e.SmartCard != "piv" {
		return fmt.Errorf("no smart-card pattern supports --smartcard %q; the OIDC patterns require cac or piv", e.SmartCard)
	}
	// idp and smart card are otherwise valid (env.Validate guards the enums),
	// so the remaining cause is the OpenShift version: external OIDC is 4.20+.
	return fmt.Errorf("no auth pattern matches (idp=%s, smartcard=%s, ocp=%s); supported identity providers: %s; requires OpenShift 4.20+",
		e.IDP, e.SmartCard, e.OCPVersion, strings.Join(env.AllowedIDPs(), ", "))
}
