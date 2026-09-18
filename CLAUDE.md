# CLAUDE.md

Context for Claude Code. Read this first every session.

## What this is
**auth-accelerator** — a personal, open-source CLI (not a commercial product, and not
affiliated with or endorsed by Red Hat) that generates the config + runbook to enable
smart-card (CAC/PIV) auth on OpenShift **4.20+**. The admin declares what identity
components they have; the tool selects the right topology and renders the `Authentication`
CR, an IdP recipe, a runbook, and an ADR.

## Scope decisions (don't relitigate)
- **OCP 4.20+ only.** External/direct OIDC is GA there, so the only integration path is the
  `Authentication` CR (`type: OIDC`, `oidcProviders`). No legacy OAuth-server branch.
- The smart-card x509/CBA validation lives at the **IdP**; OpenShift just consumes the IdP's
  OIDC tokens.
- **Supported IdPs (each a catalog pattern):** RHBK/Keycloak (x509), Okta (CBA), Microsoft
  Entra ID (CBA), PingFederate (X.509 Integration Kit). All validate PIV/CAC at the IdP.
- **Google is intentionally NOT a pattern** — no OIDC-native PIV path (its smart-card story is
  SAML + middleware; `accounts.google.com` doesn't validate certs), so there is nothing to
  wire an `Authentication` CR to. ADFS dropped; PingOne (SaaS) deferred.

## Architecture
```
main.go              thin entrypoint: cmd.Execute()
cmd/                 cobra CLI — root, generate, list, version (flags live here)
internal/env         Environment struct, defaults, validation, huh-based Prompt.
                     OIDCDefaults + ApplyWiring carry per-IdP OIDC field values.
internal/catalog     AuthPattern[] — the knowledge base of topologies. Each pattern
                     carries its Wiring (env.OIDCDefaults).
internal/engine      Select(env) -> *AuthPattern. Pure, deterministic, unit-tested.
internal/render      go:embed templates -> artifacts. add1 funcmap registered here.
internal/render/templates/*.tmpl
```
Flow: `generate` → env (flags/prompt) → `engine.Select` → `e.ApplyWiring(pattern.Wiring)`
→ `render.Generate`. Adding an IdP = one catalog entry (incl. `Wiring`) + one recipe
template; the shared templates stay generic and read `.Env.*`. A pattern with empty
`CABundleConfigMap` (public-CA SaaS: Entra/Okta) makes the CR omit
`issuerCertificateAuthority`; self-hosted IdPs (RHBK, PingFederate) set it.

## Hard rules
- **The decision engine stays deterministic.** No LLM in `engine`/`catalog`. The value is the
  encoded expertise, not a model call.
- **The tool never mutates a cluster on its own.** It generates files. `--from-cluster` (to add)
  is read-only introspection. Applying is the admin's deliberate step.
- **Break-glass before apply, always.** Enabling external OIDC removes the OAuth server; every
  generated runbook must lead with saving a break-glass kubeconfig.
- **`catalog` is where the value lives.** New capabilities = new `AuthPattern` entries, not
  new plumbing.
- Keep facts version-accurate (4.20 GA; only one OIDC provider allowed; Keycloak doesn't
  auto-provision cert→user mapping; CAC identity is the EDIPI in the SAN).
- Dependencies are **vendored** (`vendor/` — cobra, huh/bubbletea/lipgloss, yaml.v3) so the
  build stays offline / air-gap friendly.

## Commands
```bash
go build ./...
go test ./...
go run . list
go run . generate --idp <rhbk|keycloak|okta|entra|ping> --smartcard <cac|piv> --issuer <url> --out ./out
go run . generate --interactive
```
See `README.md` for the full flag reference and per-IdP examples.

## Next steps (the upgrade seams, in order)
1. ~~Swap flag dispatch for **cobra** under `cmd/`~~ (done).
2. ~~Swap `env.Prompt` for **charmbracelet/huh** (the "select what you have" wizard)~~ (done).
3. Add **`--from-cluster`** via **client-go**: confirm 4.20+, detect RHBK, read the existing
   `Authentication` CR, pre-flight prereqs (trust bundle, CRL/OCSP reachability, break-glass).
4. ~~IdP patterns: RHBK, Okta, Entra, PingFederate~~ (done via the `Wiring` seam). Google
   excluded (see Scope). Possible future: PingOne (SaaS).
5. ~~Package: static binary + UBI image to `quay.io/cmalafis10/...`~~ (done, via goreleaser).

## Working style
Brief before building, propose a plan for multi-file changes, show diffs, and run
`go build ./... && go test ./...` before declaring anything done.
