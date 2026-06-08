# RHBK / Keycloak smart-card (x509) recipe — <DATE>

Identity provider: **rhbk** · Smart card: **cac** · PKI: **dod**

The smart-card authentication happens entirely at the IdP. OpenShift then trusts
the tokens it issues via the generated `authentication-cr.yaml`.

## 1. Mutual TLS / ingress
RHBK must terminate client-certificate mTLS itself, so expose it through
**passthrough ingress** (not edge/reencrypt at the router).

## 2. Trust the dod PKI
Import the full chain (root + intermediates) into the realm truststore so the
authenticator can validate presented client certificates.

## 3. Browser flow
Duplicate the realm's `browser` flow and add an **X509/Validate Username Form**
execution set to **Alternative**, with certificate-validity checking enabled.

## 4. Certificate → user mapping
rhbk does not auto-provision identities. Map the certificate identity to
an existing user attribute that backs `preferred_username`:
- For **CAC**, read the **EDIPI/UPN from the certificate SAN**, not the Subject CN.
- Group source for this deployment: **claims**.

## 5. OIDC clients
- Confidential client `openshift-console` — redirect `https://<console>/auth/callback`.
- Public client `openshift-cli` — redirect `http://localhost:8080`.
- Ensure both are emitted in the `groups` and `preferred_username` claims.

> Issuer URL wired into OpenShift: `https://sso.example.mil/realms/fed`
