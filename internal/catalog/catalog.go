// Package catalog is the knowledge base: the typed, version-accurate auth
// topologies. THIS IS THE IP. The decision engine and renderer are plumbing;
// the value lives in these patterns. Add entries here as you encode more paths.
package catalog

import (
	"strconv"
	"strings"

	"github.com/cmalafis10/auth-accelerator/internal/env"
)

// ManifestSpec maps an embedded template to its output filename.
type ManifestSpec struct {
	Template string // template name under internal/render/templates
	Out      string // output filename written to the --out directory
}

// AuthPattern is one encoded integration topology.
type AuthPattern struct {
	ID             string
	Title          string
	IDP            string
	OCPIntegration string // external-oidc (only path we target for 4.20+)
	MinOCPVersion  string // e.g. "4.20"

	// Applies decides whether this pattern fits the given environment.
	Applies func(env.Environment) bool

	Prereqs    []string
	Steps      []string // ordered runbook
	Gotchas    []string // the things that waste a day
	BreakGlass string   // mandatory safety step before applying
	Manifests  []ManifestSpec
}

// Patterns is the ordered knowledge base. The engine returns the first match.
var Patterns = []AuthPattern{
	{
		ID:             "rhbk-cac-external-oidc",
		Title:          "RHBK/Keycloak CAC/PIV smart card → OpenShift 4.20+ external OIDC",
		IDP:            "rhbk",
		OCPIntegration: "external-oidc",
		MinOCPVersion:  "4.20",
		Applies: func(e env.Environment) bool {
			idpOK := e.IDP == "rhbk" || e.IDP == "keycloak"
			cardOK := e.SmartCard == "cac" || e.SmartCard == "piv"
			return idpOK && cardOK && versionAtLeast(e.OCPVersion, "4.20")
		},
		Prereqs: []string{
			"OpenShift 4.20+ (external/direct OIDC is GA in 4.20).",
			"Red Hat build of Keycloak (RHBK) reachable from the cluster and from clients.",
			"DoD PKI trust chain (root + intermediates) available as a CA bundle file.",
			"RHBK fronted by passthrough ingress so client-cert mTLS terminates at Keycloak, not the router.",
		},
		Steps: []string{
			"Save a break-glass kubeconfig and store it OFF-cluster before changing anything.",
			"On RHBK: import the DoD PKI chain into the realm truststore and enable mutual TLS.",
			"On RHBK: duplicate the browser flow, add 'X509/Validate Username Form' as an Alternative step, enable certificate-validity checking.",
			"On RHBK: map the certificate identity (EDIPI from the cert SAN, or CN) to the user attribute used for preferred_username.",
			"On RHBK: create OIDC clients — confidential 'openshift-console' and public 'openshift-cli' — with correct redirect URIs.",
			"Create the issuer CA bundle configmap in openshift-config (see runbook command).",
			"Apply the generated Authentication CR.",
			"Watch the rollout until authentication.config/cluster reports type: OIDC and kube-apiserver finishes its revision.",
			"Validate: oc login via the oc-oidc plugin with a CAC inserted; confirm group claims map to the expected RBAC.",
		},
		Gotchas: []string{
			"Enabling external OIDC removes the OpenShift OAuth server — without a saved break-glass kubeconfig you can lock yourself out of the cluster.",
			"Only one OIDC provider is allowed cluster-wide.",
			"Keycloak does NOT auto-provision cert→user; the mapped claim must match an existing user attribute (federate from LDAP/AD or pre-provision users).",
			"DoD CAC SANs carry the EDIPI/UPN — map the correct SAN field, not the Subject CN.",
			"Disconnected/air-gapped: confirm CRL/OCSP endpoints are reachable or stand up a local CRL distribution point.",
			"FIPS mode: some smart cards negotiate only TLS 1.2 — align RHBK and cluster cipher config.",
		},
		BreakGlass: "oc config view --flatten > break-glass.kubeconfig   # store securely OFF-cluster before applying",
		Manifests: []ManifestSpec{
			{Template: "authentication-cr.yaml.tmpl", Out: "authentication-cr.yaml"},
			{Template: "keycloak-x509-recipe.md.tmpl", Out: "keycloak-x509-recipe.md"},
			{Template: "runbook.md.tmpl", Out: "runbook.md"},
			{Template: "adr.md.tmpl", Out: "adr-0001-smart-card-auth.md"},
		},
	},
}

// --- version helper (kept here because Applies depends on it; no import cycle) ---

func versionAtLeast(got, want string) bool {
	gM, gm := parseMajorMinor(got)
	wM, wm := parseMajorMinor(want)
	if gM != wM {
		return gM > wM
	}
	return gm >= wm
}

func parseMajorMinor(v string) (int, int) {
	parts := strings.SplitN(strings.TrimSpace(v), ".", 3)
	maj, min := 0, 0
	if len(parts) > 0 {
		maj, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		min, _ = strconv.Atoi(parts[1])
	}
	return maj, min
}
