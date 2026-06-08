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

	// Wiring is the IdP-appropriate OIDC field set the CLI applies to the
	// Environment after this pattern is selected (provider name, claim names,
	// client wiring). This is how each IdP renders its own values.
	Wiring env.OIDCDefaults

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
		Wiring: env.OIDCDefaults{
			ProviderName:      "rhbk-cac",
			ConsoleClientID:   "openshift-console",
			CLIClientID:       "openshift-cli",
			ConsoleSecretName: "console-oidc-secret",
			UsernameClaim:     "preferred_username",
			GroupsClaim:       "groups",
			CABundleConfigMap: "oidc-ca-bundle",
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
	{
		ID:             "okta-cba-external-oidc",
		Title:          "Okta CBA PIV/CAC smart card → OpenShift 4.20+ external OIDC",
		IDP:            "okta",
		OCPIntegration: "external-oidc",
		MinOCPVersion:  "4.20",
		Applies: func(e env.Environment) bool {
			cardOK := e.SmartCard == "cac" || e.SmartCard == "piv"
			return e.IDP == "okta" && cardOK && versionAtLeast(e.OCPVersion, "4.20")
		},
		Wiring: env.OIDCDefaults{
			ProviderName:      "okta-cba",
			ConsoleClientID:   "openshift-console",
			CLIClientID:       "openshift-cli",
			ConsoleSecretName: "console-oidc-secret",
			UsernameClaim:     "email",
			GroupsClaim:       "groups",
			CABundleConfigMap: "oidc-ca-bundle",
		},
		Prereqs: []string{
			"OpenShift 4.20+ (external/direct OIDC is GA in 4.20).",
			"Okta org on the Identity Engine — Certificate-Based Authentication (CBA) is an OIE feature; Okta Classic cannot do PIV.",
			"DoD/federal PKI trust anchors (root + intermediates) uploaded to Okta's CBA configuration.",
			"Console and CLI OIDC apps created in Okta (confidential web app + native/public client).",
			"Cluster and clients have network egress to the Okta org (Okta is SaaS — not reachable from a fully air-gapped cluster).",
		},
		Steps: []string{
			"Save a break-glass kubeconfig and store it OFF-cluster before changing anything.",
			"In Okta: create the OIDC apps — confidential web app 'openshift-console' (redirect https://<console>/auth/callback) and native/public client 'openshift-cli' (redirect http://localhost:8080).",
			"In Okta: enable the Smart Card (PIV) authenticator under Certificate-Based Authentication and upload the DoD/federal root + intermediate CAs.",
			"In Okta: add an authentication policy (or global session/routing rule) that requires the Smart Card authenticator for the OpenShift apps.",
			"In Okta: map the certificate identity (EDIPI/UPN from the cert SAN) to the user attribute backing the email/username claim, and add a filtered 'groups' claim to the ID/access token.",
			"Create the console client secret in openshift-config (see runbook command) — the CR references it.",
			"Create the issuer CA bundle configmap in openshift-config (see runbook command).",
			"Apply the generated Authentication CR.",
			"Watch the rollout until authentication.config/cluster reports type: OIDC and kube-apiserver finishes its revision.",
			"Validate: oc login via the oc-oidc plugin with a PIV card inserted; confirm group claims map to the expected RBAC.",
		},
		Gotchas: []string{
			"Enabling external OIDC removes the OpenShift OAuth server — without a saved break-glass kubeconfig you can lock yourself out of the cluster.",
			"Only one OIDC provider is allowed cluster-wide.",
			"Okta does not emit a groups claim by default — add it explicitly with a group filter or RBAC will see no groups.",
			"CBA/PIV requires the Okta Identity Engine; an Okta Classic org cannot present the smart-card authenticator.",
			"DoD CAC/PIV SANs carry the EDIPI/UPN — map the correct SAN field, not the Subject CN.",
			"The console client secret must exist in openshift-config before the CR reconciles, or the web console stays down.",
			"Okta is SaaS: a fully disconnected/air-gapped cluster cannot reach it — this pattern needs egress to the Okta org.",
			"FIPS mode: some smart cards negotiate only TLS 1.2 — align cluster cipher config accordingly.",
		},
		BreakGlass: "oc config view --flatten > break-glass.kubeconfig   # store securely OFF-cluster before applying",
		Manifests: []ManifestSpec{
			{Template: "authentication-cr.yaml.tmpl", Out: "authentication-cr.yaml"},
			{Template: "okta-cba-recipe.md.tmpl", Out: "okta-cba-recipe.md"},
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
