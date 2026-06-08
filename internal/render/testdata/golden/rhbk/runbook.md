# Runbook — RHBK/Keycloak CAC/PIV smart card → OpenShift 4.20+ external OIDC

Generated <DATE> by auth-accelerator · pattern `rhbk-cac-external-oidc` · integration `external-oidc` (OCP 4.20+)

## ⚠️ Break-glass first
```
oc config view --flatten > break-glass.kubeconfig   # store securely OFF-cluster before applying
```

## Prerequisites
- OpenShift 4.20+ (external/direct OIDC is GA in 4.20).
- Red Hat build of Keycloak (RHBK) reachable from the cluster and from clients.
- DoD PKI trust chain (root + intermediates) available as a CA bundle file.
- RHBK fronted by passthrough ingress so client-cert mTLS terminates at Keycloak, not the router.

## Steps
1. Save a break-glass kubeconfig and store it OFF-cluster before changing anything.
2. On RHBK: import the DoD PKI chain into the realm truststore and enable mutual TLS.
3. On RHBK: duplicate the browser flow, add 'X509/Validate Username Form' as an Alternative step, enable certificate-validity checking.
4. On RHBK: map the certificate identity (EDIPI from the cert SAN, or CN) to the user attribute used for preferred_username.
5. On RHBK: create OIDC clients — confidential 'openshift-console' and public 'openshift-cli' — with correct redirect URIs.
6. Create the issuer CA bundle configmap in openshift-config (see runbook command).
7. Apply the generated Authentication CR.
8. Watch the rollout until authentication.config/cluster reports type: OIDC and kube-apiserver finishes its revision.
9. Validate: oc login via the oc-oidc plugin with a CAC inserted; confirm group claims map to the expected RBAC.

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
oc login   # uses the oc-oidc exec plugin against https://sso.example.mil/realms/fed
```

## Username mapping → RBAC
This deployment maps the OpenShift username from the `preferred_username` claim. Because
no username prefix is configured (`prefixPolicy: NoOpinion`), OpenShift may prefix a username
that does not come from the `sub` claim with the issuer URL — e.g. `https://sso.example.mil/realms/fed#<value>`.
After the first login, run `oc whoami` to see the exact username string, and make your
`RoleBinding` / `ClusterRoleBinding` subjects match it.

## Gotchas (read before you start)
- Enabling external OIDC removes the OpenShift OAuth server — without a saved break-glass kubeconfig you can lock yourself out of the cluster.
- Only one OIDC provider is allowed cluster-wide.
- Keycloak does NOT auto-provision cert→user; the mapped claim must match an existing user attribute (federate from LDAP/AD or pre-provision users).
- DoD CAC SANs carry the EDIPI/UPN — map the correct SAN field, not the Subject CN.
- Disconnected/air-gapped: confirm CRL/OCSP endpoints are reachable or stand up a local CRL distribution point.
- FIPS mode: some smart cards negotiate only TLS 1.2 — align RHBK and cluster cipher config.

