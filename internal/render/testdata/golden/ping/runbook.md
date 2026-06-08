# Runbook — PingFederate X.509 PIV/CAC smart card → OpenShift 4.20+ external OIDC

Generated <DATE> by auth-accelerator · pattern `ping-cac-external-oidc` · integration `external-oidc` (OCP 4.20+)

## ⚠️ Break-glass first
```
oc config view --flatten > break-glass.kubeconfig   # store securely OFF-cluster before applying
```

## Prerequisites
- OpenShift 4.20+ (external/direct OIDC is GA in 4.20).
- PingFederate reachable from the cluster and from clients.
- PingFederate X.509 Certificate Integration Kit installed — it is an add-on, not a built-in adapter.
- DoD/federal PKI trust chain (root + intermediates) available as a CA bundle file.
- PingFederate fronted by passthrough ingress so client-cert mTLS terminates at PingFederate, not the router.

## Steps
1. Save a break-glass kubeconfig and store it OFF-cluster before changing anything.
2. On PingFederate: install the X.509 Certificate Integration Kit and configure the X.509 IdP Adapter — set acceptable issuers to the DoD/federal PKI and extract the EDIPI/UPN from the certificate SAN.
3. On PingFederate: require client certificates (enable mutual TLS) via the passthrough ingress so the card is challenged at sign-in.
4. On PingFederate: map the extracted SAN identity to the attribute backing preferred_username.
5. On PingFederate: create OIDC clients — confidential 'openshift-console' (redirect https://<console>/auth/callback) and public 'openshift-cli' (redirect http://localhost:8080).
6. On PingFederate: extend the access-token contract / OIDC policy so the username and 'groups' claims are emitted to OpenShift.
7. Create the console client secret in openshift-config (see runbook command) — the CR references it.
8. Create the issuer CA bundle configmap in openshift-config (see runbook command).
9. Apply the generated Authentication CR.
10. Watch the rollout until authentication.config/cluster reports type: OIDC and kube-apiserver finishes its revision.
11. Validate: oc login via the oc-oidc plugin with a CAC/PIV card inserted; confirm group claims map to the expected RBAC.

### Key commands
```
# Create the issuer CA bundle configmap (dod PKI chain):
oc create configmap oidc-ca-bundle \
  --from-file=ca-bundle.crt=dod-pki-chain.crt -n openshift-config

# Create the console client secret (confidential client):
oc create secret generic console-oidc-secret \
  --from-literal=clientSecret=<console-client-secret> -n openshift-config

# Apply the generated Authentication CR:
oc apply -f authentication-cr.yaml

# Watch the rollout:
oc get authentication.config/cluster -o jsonpath='{.spec.type}{"\n"}'   # -> OIDC
oc get clusteroperator kube-apiserver

# Validate with a cac smart card inserted:
oc login   # uses the oc-oidc exec plugin against https://pingfed.example.mil
```

## Username mapping → RBAC
This deployment maps the OpenShift username from the `preferred_username` claim. Because
no username prefix is configured (`prefixPolicy: NoOpinion`), OpenShift may prefix a username
that does not come from the `sub` claim with the issuer URL — e.g. `https://pingfed.example.mil#<value>`.
After the first login, run `oc whoami` to see the exact username string, and make your
`RoleBinding` / `ClusterRoleBinding` subjects match it.

## Gotchas (read before you start)
- Enabling external OIDC removes the OpenShift OAuth server — without a saved break-glass kubeconfig you can lock yourself out of the cluster.
- Only one OIDC provider is allowed cluster-wide.
- The X.509 Certificate Integration Kit is an add-on you must install — a stock PingFederate has no certificate adapter.
- DoD CAC/PIV SANs carry the EDIPI/UPN — map the correct SAN field, not the Subject CN.
- PingFederate emits no groups by default — extend the access-token contract / OIDC policy to include the 'groups' claim or RBAC sees none.
- Certificate validation needs CRL/OCSP reachability — air-gapped: stand up a local CRL distribution point.
- The console client secret must exist in openshift-config before the CR reconciles, or the web console stays down.
- FIPS mode: some smart cards negotiate only TLS 1.2 — align PingFederate and cluster cipher config.

