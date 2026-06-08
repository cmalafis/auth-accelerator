# Runbook — Okta CBA PIV/CAC smart card → OpenShift 4.20+ external OIDC

Generated <DATE> by auth-accelerator · pattern `okta-cba-external-oidc` · integration `external-oidc` (OCP 4.20+)

## ⚠️ Break-glass first
```
oc config view --flatten > break-glass.kubeconfig   # store securely OFF-cluster before applying
```

## Prerequisites
- OpenShift 4.20+ (external/direct OIDC is GA in 4.20).
- Okta org on the Identity Engine — Certificate-Based Authentication (CBA) is an OIE feature; Okta Classic cannot do PIV.
- DoD/federal PKI trust anchors (root + intermediates) uploaded to Okta's CBA configuration.
- Console and CLI OIDC apps created in Okta (confidential web app + native/public client).
- Cluster and clients have network egress to the Okta org (Okta is SaaS — not reachable from a fully air-gapped cluster).

## Steps
1. Save a break-glass kubeconfig and store it OFF-cluster before changing anything.
2. In Okta: create the OIDC apps — confidential web app 'openshift-console' (redirect https://<console>/auth/callback) and native/public client 'openshift-cli' (redirect http://localhost:8080).
3. In Okta: enable the Smart Card (PIV) authenticator under Certificate-Based Authentication and upload the DoD/federal root + intermediate CAs.
4. In Okta: add an authentication policy (or global session/routing rule) that requires the Smart Card authenticator for the OpenShift apps.
5. In Okta: map the certificate identity (EDIPI/UPN from the cert SAN) to the user attribute backing the email/username claim, and add a filtered 'groups' claim to the ID/access token.
6. Create the console client secret in openshift-config (see runbook command) — the CR references it.
7. Create the issuer CA bundle configmap in openshift-config (see runbook command).
8. Apply the generated Authentication CR.
9. Watch the rollout until authentication.config/cluster reports type: OIDC and kube-apiserver finishes its revision.
10. Validate: oc login via the oc-oidc plugin with a PIV card inserted; confirm group claims map to the expected RBAC.

### Key commands
```
# Create the issuer CA bundle configmap (federal PKI chain):
oc create configmap oidc-ca-bundle \
  --from-file=ca-bundle.crt=federal-pki-chain.crt -n openshift-config

# Create the console client secret (confidential client):
oc create secret generic console-oidc-secret \
  --from-literal=clientSecret=<console-client-secret> -n openshift-config

# Apply the generated Authentication CR:
oc apply -f authentication-cr.yaml

# Watch the rollout:
oc get authentication.config/cluster -o jsonpath='{.spec.type}{"\n"}'   # -> OIDC
oc get clusteroperator kube-apiserver

# Validate with a piv smart card inserted:
oc login   # uses the oc-oidc exec plugin against https://example.okta.com
```

## Username mapping → RBAC
This deployment maps the OpenShift username from the `email` claim. Because
no username prefix is configured (`prefixPolicy: NoOpinion`), OpenShift may prefix a username
that does not come from the `sub` claim with the issuer URL — e.g. `https://example.okta.com#<value>`.
After the first login, run `oc whoami` to see the exact username string, and make your
`RoleBinding` / `ClusterRoleBinding` subjects match it.

## Gotchas (read before you start)
- Enabling external OIDC removes the OpenShift OAuth server — without a saved break-glass kubeconfig you can lock yourself out of the cluster.
- Only one OIDC provider is allowed cluster-wide.
- Okta does not emit a groups claim by default — add it explicitly with a group filter or RBAC will see no groups.
- CBA/PIV requires the Okta Identity Engine; an Okta Classic org cannot present the smart-card authenticator.
- DoD CAC/PIV SANs carry the EDIPI/UPN — map the correct SAN field, not the Subject CN.
- The console client secret must exist in openshift-config before the CR reconciles, or the web console stays down.
- Okta is SaaS: a fully disconnected/air-gapped cluster cannot reach it — this pattern needs egress to the Okta org.
- FIPS mode: some smart cards negotiate only TLS 1.2 — align cluster cipher config accordingly.

