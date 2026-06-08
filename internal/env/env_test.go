package env

import "testing"

func TestValidate_DefaultsPass(t *testing.T) {
	if err := Defaults().Validate(); err != nil {
		t.Fatalf("Defaults() should validate, got: %v", err)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	cases := map[string]func(*Environment){
		"missing ocp":    func(e *Environment) { e.OCPVersion = "" },
		"missing idp":    func(e *Environment) { e.IDP = "" },
		"missing issuer": func(e *Environment) { e.IssuerURL = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := Defaults()
			mutate(&e)
			if err := e.Validate(); err == nil {
				t.Fatalf("expected error for %s, got nil", name)
			}
		})
	}
}

func TestValidate_RejectsBadEnums(t *testing.T) {
	cases := map[string]func(*Environment){
		"idp":       func(e *Environment) { e.IDP = "okra" },
		"smartcard": func(e *Environment) { e.SmartCard = "rfid" },
		"pki":       func(e *Environment) { e.PKI = "nato" },
		"flavor":    func(e *Environment) { e.OCPFlavor = "k3s" },
		"groups":    func(e *Environment) { e.GroupSource = "telepathy" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			e := Defaults()
			mutate(&e)
			if err := e.Validate(); err == nil {
				t.Fatalf("expected error for bad %s, got nil", name)
			}
		})
	}
}

func TestValidate_AcceptsEveryAllowedIDP(t *testing.T) {
	for _, idp := range allowedIDP {
		e := Defaults()
		e.IDP = idp
		if err := e.Validate(); err != nil {
			t.Errorf("idp %q should be valid: %v", idp, err)
		}
	}
}

func TestValidate_IssuerURL(t *testing.T) {
	good := []string{
		"https://sso.example.mil/realms/fed",
		"https://login.microsoftonline.com/tenant/v2.0",
		"https://example.okta.com",
	}
	for _, u := range good {
		e := Defaults()
		e.IssuerURL = u
		if err := e.Validate(); err != nil {
			t.Errorf("issuer %q should be valid: %v", u, err)
		}
	}

	bad := []string{
		"http://insecure.example.mil", // not https
		"sso.example.mil",             // no scheme/host
		"https://",                    // missing host
		"not a url",                   // malformed
	}
	for _, u := range bad {
		e := Defaults()
		e.IssuerURL = u
		if err := e.Validate(); err == nil {
			t.Errorf("issuer %q should be rejected, got nil", u)
		}
	}
}
