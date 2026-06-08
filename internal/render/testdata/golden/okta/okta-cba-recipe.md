# Okta Certificate-Based Authentication (CBA) recipe — <DATE>

Identity provider: **okta** · Smart card: **piv** · PKI: **federal**

The smart-card authentication happens entirely at Okta via **Certificate-Based
Authentication (CBA)**. OpenShift then trusts the OIDC tokens Okta issues, via the
generated `authentication-cr.yaml`. CBA is an **Okta Identity Engine** feature — an
Okta Classic org cannot present the PIV authenticator.

## 1. OIDC applications
- Confidential web app `openshift-console` — sign-in redirect
  `https://<console>/auth/callback`. Note its **client secret**; it goes into the
  `console-oidc-secret` secret in `openshift-config`.
- Native / public client `openshift-cli` — redirect `http://localhost:8080`
  (used by the `oc-oidc` exec plugin; no secret).

## 2. Smart Card (PIV) authenticator
Under **Security → Authenticators**, add the **Smart Card / PIV** authenticator and
upload the **federal** trust chain (root + intermediates) so Okta can validate
presented client certificates.

## 3. Authentication policy
Add an authentication policy (or global session / routing rule) that **requires the
Smart Card authenticator** for the OpenShift apps above, so users must present a
piv card rather than a password.

## 4. Certificate → user mapping
Okta matches the presented certificate to an existing user. Map the certificate
identity to the attribute backing `email`:
- Map the appropriate SAN field (EDIPI/UPN) to the user attribute, not the Subject CN.
- Group source for this deployment: **claims**.

## 5. Claims
- Add a **`groups`** claim to the ID/access token with a group filter —
  Okta emits no groups claim by default, and OpenShift RBAC will see no groups without it.
- Ensure the **`email`** claim is emitted for the username mapping.

> Issuer URL wired into OpenShift: `https://example.okta.com`
