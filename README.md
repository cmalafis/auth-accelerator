# auth-accelerator

An internal Red Hat accelerator that generates the YAML and step-by-step runbook to
enable **smart-card (CAC/PIV) authentication on OpenShift 4.20+**, based on the
identity components an admin has available. The sysadmin says what they have; the
tool emits the `Authentication` CR, the IdP recipe, a runbook, and an ADR.

Scope is deliberately **OCP 4.20+ only**, where external/direct OIDC is GA — so there's
a single clean integration path (the `Authentication` CR with `oidcProviders`), no
legacy OAuth branch.

## Build & run
```bash
go build ./...
go test ./...
go run . list
go run . generate --idp rhbk --smartcard cac --ocp 4.20 --issuer https://sso.example.mil/realms/fed --out ./out
go run . generate --interactive
```
Output lands in `./out`: `authentication-cr.yaml`, `keycloak-x509-recipe.md`,
`runbook.md`, and an ADR. Nothing touches a cluster — review the runbook and save a
break-glass kubeconfig before applying anything by hand.

## Architecture
```
main.go                       # stdlib flag CLI (dispatch + flags)
internal/
  env/      Environment        # the admin's answers + interactive prompt + validation
  catalog/  AuthPattern[]      # THE KNOWLEDGE BASE — the IP. Add patterns here.
  engine/   Select(env)        # pure, deterministic: pick the pattern. Unit-tested.
  render/   Generate(...)      # go:embed templates -> rendered artifacts
    templates/*.tmpl
```
The brain stays deterministic — no LLM in the decision path. The value is the encoded
expertise in `catalog`, not the plumbing.

## Built as a stdlib-only skeleton — the upgrade seams
This compiles offline with zero external dependencies. The intended next steps (your
Mac can pull these via the module proxy):
1. **cobra** — replace the flag dispatch in `main.go` with `spf13/cobra` under `cmd/`.
2. **huh** — replace `env.Prompt` with `charmbracelet/huh` for the "select what you have" wizard.
3. **client-go** — add `--from-cluster`: read `ClusterVersion` (confirm 4.20+), detect the
   RHBK operator, inspect the existing `Authentication` CR, and pre-flight prereqs
   (DoD trust bundle present? CRL/OCSP reachable? break-glass saved?).
4. **Distribution** — `go:embed` already bundles templates into the binary; ship a static
   binary for laptops/bastions and a UBI image (`quay.io/cmalafis10/...`) for air-gapped sites.
```
