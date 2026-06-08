# auth-accelerator

An internal Red Hat accelerator that generates the YAML and a step-by-step runbook to
enable **smart-card (CAC/PIV) authentication on OpenShift 4.20+**, based on the identity
components an admin already has. The admin declares what they have; the tool selects the
right topology and emits the `Authentication` CR, an IdP-specific recipe, a runbook, and an
ADR.

Scope is deliberately **OCP 4.20+ only**, where external/direct OIDC is GA — so there's a
single clean integration path (the `Authentication` CR with `oidcProviders`), no legacy
OAuth branch. The smart-card x509/CBA validation always happens **at the IdP**; OpenShift
just consumes the IdP's OIDC tokens.

> **Safety:** the tool only ever *writes files* — it never touches a cluster. Review the
> generated `runbook.md` and **save a break-glass kubeconfig before you apply anything**.
> Enabling external OIDC removes the built-in OAuth server.

## Supported identity providers

| `--idp` | Pattern | Smart-card mechanism | Issuer CA |
|---------|---------|----------------------|-----------|
| `rhbk` / `keycloak` | RHBK/Keycloak | x509 browser flow (Validate Username Form) | bundle (self-hosted) |
| `okta` | Okta | Certificate-Based Auth (Identity Engine) | omitted (public CA) |
| `entra` | Microsoft Entra ID | Certificate-Based Auth | omitted (public CA) |
| `ping` | PingFederate | X.509 Certificate Integration Kit | bundle (self-hosted) |

`adfs` and `google` are **not supported**: Google has no OIDC-native PIV path (its
smart-card story is SAML + middleware), and ADFS is out of scope. PingOne (Ping's SaaS) is
deferred in favor of self-hosted PingFederate.

## Install & build

```bash
go build ./...      # build (cobra is vendored, so this works offline / air-gapped)
go test ./...       # run the unit tests
go install .        # optional: put `auth-accelerator` on your PATH
```

Requires Go 1.22+. Dependencies are vendored under `vendor/`, so no network is needed to
build.

## Usage

```
auth-accelerator <command> [flags]

Commands:
  generate    Generate the Authentication CR + runbook for your environment
  list        List the known auth patterns
  version     Print the version
```

Run `auth-accelerator generate -h` for the full flag list. (Examples below use `go run .`;
substitute the built `auth-accelerator` binary if you ran `go install`.)

### `generate` flags

| Flag | Default | Meaning |
|------|---------|---------|
| `--idp` | `rhbk` | Identity provider: `rhbk` \| `keycloak` \| `okta` \| `entra` \| `ping` |
| `--smartcard` | `cac` | Smart card type: `cac` \| `piv` \| `eca` |
| `--issuer` | _(RHBK sample)_ | OIDC issuer URL (point this at **your** IdP) |
| `--ocp` | `4.20` | OpenShift version (must be 4.20+) |
| `--flavor` | `self-managed` | `self-managed` \| `rosa` \| `aro` \| `okd` |
| `--pki` | `dod` | PKI chain: `dod` \| `federal` \| `eca` |
| `--groups` | `claims` | Group source: `claims` \| `ldap` |
| `--fips` | `false` | Cluster runs in FIPS mode (adds cipher guidance) |
| `--air-gapped` | `false` | Disconnected environment (adds CRL/OCSP guidance) |
| `--interactive` | `false` | Prompt for inputs instead of using flags |
| `--out` | `./out` | Output directory |

The only flag you must set for a real run is `--issuer` (your IdP's OIDC issuer URL). The
provider name, claim names, and client wiring are derived from the chosen `--idp`.

### Examples

```bash
# List what the tool knows
go run . list

# RHBK / Keycloak — CAC, DoD PKI
go run . generate --idp rhbk --smartcard cac --pki dod \
  --issuer https://sso.example.mil/realms/fed --out ./out

# Okta — PIV, federal PKI
go run . generate --idp okta --smartcard piv --pki federal \
  --issuer https://example.okta.com --out ./out

# Microsoft Entra ID — CAC (issuer is the tenant's v2.0 endpoint)
go run . generate --idp entra --smartcard cac \
  --issuer https://login.microsoftonline.com/<tenant-id>/v2.0 --out ./out

# PingFederate — CAC, DoD PKI
go run . generate --idp ping --smartcard cac --pki dod \
  --issuer https://pingfed.example.mil --out ./out

# Interactive wizard (press Enter to accept each default)
go run . generate --interactive

# FIPS + air-gapped variants add extra runbook gotchas
go run . generate --idp rhbk --fips --air-gapped --issuer https://sso.example.mil/realms/fed
```

## What it generates

Into `--out` (default `./out`):

| File | Purpose |
|------|---------|
| `authentication-cr.yaml` | The OpenShift `Authentication` CR (`type: OIDC`) you apply |
| `<idp>-recipe.md` | IdP-side setup (e.g. `okta-cba-recipe.md`, `ping-x509-recipe.md`) |
| `runbook.md` | Ordered, break-glass-first runbook with the exact `oc` commands |
| `adr-0001-smart-card-auth.md` | The architecture decision record |

After generating: review `runbook.md`, then **save your break-glass kubeconfig before
applying** (`oc config view --flatten > break-glass.kubeconfig`, stored off-cluster).

## Architecture

```
main.go                       # thin entrypoint: cmd.Execute()
cmd/                          # cobra CLI — root, generate, list, version (flags live here)
internal/
  env/      Environment        # the admin's answers + interactive prompt + validation
            OIDCDefaults       # per-IdP OIDC field values, applied via ApplyWiring
  catalog/  AuthPattern[]      # THE KNOWLEDGE BASE — the IP. Add patterns here.
  engine/   Select(env)        # pure, deterministic: pick the first matching pattern
  render/   Generate(...)      # go:embed templates -> rendered artifacts
    templates/*.tmpl
```

Flow: `generate` → build `Environment` (flags or `--interactive`) → `engine.Select` →
`e.ApplyWiring(pattern.Wiring)` → `render.Generate`. The decision path stays deterministic —
no LLM. The value is the encoded expertise in `catalog`, not the plumbing.

**Adding an IdP** is one `catalog.AuthPattern` entry (including its `Wiring`) plus one recipe
template — the shared templates stay generic and read `.Env.*`. A pattern that leaves
`CABundleConfigMap` empty (public-CA SaaS like Entra/Okta) makes the CR omit
`issuerCertificateAuthority`; self-hosted IdPs (RHBK, PingFederate) set it.

## Roadmap

1. **huh** — replace `env.Prompt` with `charmbracelet/huh` for a richer "select what you
   have" wizard.
2. **client-go `--from-cluster`** — read `ClusterVersion` (confirm 4.20+), detect the RHBK
   operator, inspect the existing `Authentication` CR, and pre-flight prereqs (trust bundle
   present? CRL/OCSP reachable? break-glass saved?).
3. **Distribution** — `go:embed` already bundles templates into the binary; ship a static
   binary for laptops/bastions and a UBI image (`quay.io/cmalafis10/...`) for air-gapped
   sites.
