# PingFederate smart-card (X.509) recipe — <DATE>

Identity provider: **ping** (PingFederate) · Smart card: **cac** · PKI: **dod**

The smart-card authentication happens entirely at PingFederate via the **X.509
Certificate Integration Kit**. OpenShift then trusts the OIDC tokens PingFederate
issues, via the generated `authentication-cr.yaml`.

## 1. Mutual TLS / ingress
PingFederate must terminate client-certificate mTLS itself, so expose it through
**passthrough ingress** (not edge/reencrypt at the router).

## 2. Install the X.509 Certificate Integration Kit
The certificate adapter is an **add-on**, not built in. Install the **X.509 Certificate
Integration Kit** and deploy the **X.509 Certificate IdP Adapter** instance.

## 3. Configure the X.509 IdP Adapter
- Set the **acceptable issuers** to the **dod** chain (root + intermediates) so the
  adapter validates presented client certificates.
- Enable certificate-validity checking (CRL/OCSP).
- Extract the certificate identity for mapping:
  - For **CAC**, read the **EDIPI/UPN from the certificate SAN**, not the Subject CN.

## 4. Certificate → user mapping
Map the extracted SAN identity to the attribute that backs `preferred_username`.
- Group source for this deployment: **claims**.

## 5. OIDC clients
- Confidential client `openshift-console` — redirect `https://<console>/auth/callback`.
- Public client `openshift-cli` — redirect `http://localhost:8080`.

## 6. Token contract / OIDC policy
Extend the **access-token contract** (and OIDC policy) so PingFederate emits both the
`preferred_username` and `groups` claims to OpenShift — PingFederate
sends no groups by default.

> Issuer URL wired into OpenShift: `https://pingfed.example.mil`
