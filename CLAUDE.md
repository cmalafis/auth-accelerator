# CLAUDE.md

Context for Claude Code. Read this first every session.

## What this is
**auth-accelerator** — an internal Red Hat tool (not a commercial product) that generates
the config + runbook to enable smart-card (CAC/PIV) auth on OpenShift **4.20+**. The admin
declares what identity components they have; the tool selects the right topology and renders
the `Authentication` CR, an IdP recipe, a runbook, and an ADR.

## Scope decisions (don't relitigate)
- **OCP 4.20+ only.** External/direct OIDC is GA there, so the only integration path is the
  `Authentication` CR (`type: OIDC`, `oidcProviders`). No legacy OAuth-server branch.
- The smart-card x509 validation lives at the **IdP** (RHBK/Keycloak, Entra, Okta…); OpenShift
  just consumes the IdP's OIDC tokens.

## Architecture
```
main.go              stdlib flag CLI (to be replaced by cobra)
internal/env         Environment struct, defaults, validation, stdlib Prompt (-> huh)
internal/catalog     AuthPattern[] — THE IP. Knowledge base of topologies.
internal/engine      Select(env) -> *AuthPattern. Pure, deterministic, unit-tested.
internal/render      go:embed templates -> artifacts. add1 funcmap registered here.
internal/render/templates/*.tmpl
```

## Hard rules
- **The decision engine stays deterministic.** No LLM in `engine`/`catalog`. The value is the
  encoded expertise, not a model call.
- **The tool never mutates a cluster on its own.** It generates files. `--from-cluster` (to add)
  is read-only introspection. Applying is the admin's deliberate step.
- **Break-glass before apply, always.** Enabling external OIDC removes the OAuth server; every
  generated runbook must lead with saving a break-glass kubeconfig.
- **`catalog` is the gold.** New capabilities = new `AuthPattern` entries, not new plumbing.
- Keep facts version-accurate (4.20 GA; only one OIDC provider allowed; Keycloak doesn't
  auto-provision cert→user mapping; CAC identity is the EDIPI in the SAN).

## Commands
```bash
go build ./...
go test ./...
go run . generate --interactive
```

## Next steps (the upgrade seams, in order)
1. Swap flag dispatch for **cobra** under `cmd/`.
2. Swap `env.Prompt` for **charmbracelet/huh** (the select-what-you-have wizard).
3. Add **`--from-cluster`** via **client-go**: confirm 4.20+, detect RHBK, read the existing
   `Authentication` CR, pre-flight prereqs (trust bundle, CRL/OCSP reachability, break-glass).
4. Add the next `AuthPattern`s: Entra ID CBA, ~~Okta PIV~~ (done). Each is a catalog entry +
   templates. Per-IdP OIDC field values (provider name, claim names, client wiring) live on
   `AuthPattern.Wiring` (`env.OIDCDefaults`); the CLI applies them via `Environment.ApplyWiring`
   after `engine.Select`, so templates stay generic and read `.Env.*`.
5. Package: static binary + UBI image to `quay.io/cmalafis10/...`.

## Working style
Brief before building, propose a plan for multi-file changes, show diffs, and run
`go build ./... && go test ./...` before declaring anything done.
