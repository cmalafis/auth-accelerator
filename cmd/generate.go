package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cmalafis10/auth-accelerator/internal/engine"
	"github.com/cmalafis10/auth-accelerator/internal/env"
	"github.com/cmalafis10/auth-accelerator/internal/render"
)

func newGenerateCmd() *cobra.Command {
	// Seed the Environment from Defaults so each flag's default matches today's behavior.
	e := env.Defaults()
	var (
		interactive bool
		out         string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate YAML + runbook for your environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if interactive {
				e = env.Prompt(os.Stdin)
			}
			if err := e.Validate(); err != nil {
				return err
			}

			pattern, err := engine.Select(e)
			if err != nil {
				return err
			}

			// Apply the matched pattern's IdP-appropriate OIDC wiring (provider
			// name, claim names, client wiring) before rendering.
			e.ApplyWiring(pattern.Wiring)

			written, err := render.Generate(pattern, e, out)
			if err != nil {
				return err
			}

			fmt.Printf("Matched pattern: %s\n", pattern.Title)
			fmt.Printf("Wrote %d files to %s:\n", len(written), out)
			for _, w := range written {
				fmt.Println("  -", w)
			}
			fmt.Println("\nNext: review runbook.md, then save your break-glass kubeconfig BEFORE applying.")
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&interactive, "interactive", false, "prompt for inputs instead of using flags")
	f.StringVar(&out, "out", "./out", "output directory")
	f.StringVar(&e.OCPVersion, "ocp", e.OCPVersion, "OpenShift version (e.g. 4.20)")
	f.StringVar(&e.OCPFlavor, "flavor", e.OCPFlavor, "self-managed|rosa|aro|okd")
	f.StringVar(&e.IDP, "idp", e.IDP, "rhbk|keycloak|entra|okta|ping|adfs")
	f.StringVar(&e.SmartCard, "smartcard", e.SmartCard, "cac|piv|eca")
	f.StringVar(&e.PKI, "pki", e.PKI, "dod|federal|eca")
	f.StringVar(&e.GroupSource, "groups", e.GroupSource, "claims|ldap")
	f.StringVar(&e.IssuerURL, "issuer", e.IssuerURL, "OIDC issuer URL")
	f.BoolVar(&e.FIPS, "fips", e.FIPS, "cluster runs in FIPS mode")
	f.BoolVar(&e.AirGapped, "air-gapped", e.AirGapped, "disconnected environment")

	return cmd
}
