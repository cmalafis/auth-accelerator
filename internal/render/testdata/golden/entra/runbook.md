# Runbook — Microsoft Entra ID CBA PIV/CAC smart card → OpenShift 4.20+ external OIDC

Generated <DATE> by auth-accelerator · pattern `entra-cba-external-oidc` · integration `external-oidc` (OCP 4.20+)

## ⚠️ Break-glass first
```
oc config view --flatten > break-glass.kubeconfig   # store securely OFF-cluster before applying
```

## Prerequisites
- OpenShift 4.20+ (external/direct OIDC is GA in 4.20).
- A Microsoft Entra ID tenant (commercial or Azure Government for DoD/federal).
- Entra Certificate-Based Authentication (CBA) configured with the DoD/federal CA chain uploaded to the tenant's certificate authorities.
- App registrations created for the console (confidential web app) and CLI (public/native client).
- Cluster and clients have network egress to login.microsoftonline.com (or login.microsoftonline.us) — Entra is SaaS, not reachable from a fully air-gapped cluster.

## Steps
1. Save a break-glass kubeconfig and store it OFF-cluster before changing anything.
2. In Entra: create the app registrations — confidential web app 'openshift-console' (redirect https://<console>/auth/callback, with a client secret) and public/native client 'openshift-cli' (redirect http://localhost:8080).
3. In Entra: under Authentication methods, enable Certificate-Based Authentication and upload the DoD/federal root + intermediate CAs to the tenant.
4. In Entra: set the CBA authentication binding (single/multi-factor) and the username binding — by default SAN PrincipalName maps to userPrincipalName; for high affinity use IssuerAndSerialNumber.
5. In Entra: configure the app's token configuration to emit the 'groups' claim, and confirm the username claim (email/UPN) is populated for cert users.
6. Create the console client secret in openshift-config (see runbook command) — the CR references it.
7. Apply the generated Authentication CR (no issuer CA configmap needed — Entra uses a public CA).
8. Watch the rollout until authentication.config/cluster reports type: OIDC and kube-apiserver finishes its revision.
9. Validate: oc login via the oc-oidc plugin with a PIV/CAC card inserted; confirm group claims map to the expected RBAC.

### Key commands
```
# Create the console client secret (confidential client):
oc create secret generic console-oidc-secret \
  --from-literal=clientSecret=<console-client-secret> -n openshift-config

# Apply the generated Authentication CR:
oc apply -f authentication-cr.yaml

# Watch the rollout:
oc get authentication.config/cluster -o jsonpath='{.spec.type}{"\n"}'   # -> OIDC
oc get clusteroperator kube-apiserver

# Validate with a cac smart card inserted:
oc login   # uses the oc-oidc exec plugin against https://login.microsoftonline.com/TENANT/v2.0
```

## Username mapping → RBAC
This deployment maps the OpenShift username from the `email` claim. Because
no username prefix is configured (`prefixPolicy: NoOpinion`), OpenShift may prefix a username
that does not come from the `sub` claim with the issuer URL — e.g. `https://login.microsoftonline.com/TENANT/v2.0#<value>`.
After the first login, run `oc whoami` to see the exact username string, and make your
`RoleBinding` / `ClusterRoleBinding` subjects match it.

## Gotchas (read before you start)
- Enabling external OIDC removes the OpenShift OAuth server — without a saved break-glass kubeconfig you can lock yourself out of the cluster.
- Only one OIDC provider is allowed cluster-wide.
- Entra emits group OBJECT IDs (GUIDs), not names, by default — RBAC RoleBindings must reference the group GUIDs unless you configure group-name claims (which require on-prem-synced groups).
- The username binding must agree with the username claim: CBA binds the cert SAN PrincipalName to userPrincipalName; if 'email' is not populated for cert users, map the username claim to UPN instead.
- DoD CAC/PIV SANs carry the EDIPI/UPN — ensure the certificateUserIds / username binding targets the correct field, not the Subject CN.
- The console client secret must exist in openshift-config before the CR reconciles, or the web console stays down.
- Entra is SaaS: a fully disconnected/air-gapped cluster cannot reach it — this pattern needs egress to the Entra login endpoint.
- FIPS mode: some smart cards negotiate only TLS 1.2 — align cluster cipher config accordingly.

