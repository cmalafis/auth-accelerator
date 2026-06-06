// Command auth-accelerator generates the YAML and runbook to enable smart-card
// authentication on an OpenShift 4.20+ cluster, based on the components the admin
// has available.
//
// This skeleton uses only the standard library so it builds offline. The marked
// upgrade seams for the first Claude Code session:
//   - swap this flag dispatch for spf13/cobra (cmd/ package)
//   - swap env.Prompt for charmbracelet/huh (the "select what you have" wizard)
//   - add a --from-cluster path using k8s.io/client-go to introspect the cluster
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cmalafis10/auth-accelerator/internal/catalog"
	"github.com/cmalafis10/auth-accelerator/internal/engine"
	"github.com/cmalafis10/auth-accelerator/internal/env"
	"github.com/cmalafis10/auth-accelerator/internal/render"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "generate":
		cmdGenerate(os.Args[2:])
	case "list":
		cmdList()
	case "version":
		fmt.Println("auth-accelerator", version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `auth-accelerator — smart-card auth setup for OpenShift 4.20+

Usage:
  auth-accelerator generate [flags]   Generate YAML + runbook for your environment
  auth-accelerator list               List the known auth patterns
  auth-accelerator version

Run "auth-accelerator generate -h" for flags.`)
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	interactive := fs.Bool("interactive", false, "prompt for inputs instead of using flags")
	out := fs.String("out", "./out", "output directory")

	e := env.Defaults()
	fs.StringVar(&e.OCPVersion, "ocp", e.OCPVersion, "OpenShift version (e.g. 4.20)")
	fs.StringVar(&e.OCPFlavor, "flavor", e.OCPFlavor, "self-managed|rosa|aro|okd")
	fs.StringVar(&e.IDP, "idp", e.IDP, "rhbk|keycloak|entra|okta|ping|adfs")
	fs.StringVar(&e.SmartCard, "smartcard", e.SmartCard, "cac|piv|eca")
	fs.StringVar(&e.PKI, "pki", e.PKI, "dod|federal|eca")
	fs.StringVar(&e.GroupSource, "groups", e.GroupSource, "claims|ldap")
	fs.StringVar(&e.IssuerURL, "issuer", e.IssuerURL, "OIDC issuer URL")
	fs.BoolVar(&e.FIPS, "fips", e.FIPS, "cluster runs in FIPS mode")
	fs.BoolVar(&e.AirGapped, "air-gapped", e.AirGapped, "disconnected environment")
	_ = fs.Parse(args)

	if *interactive {
		e = env.Prompt(os.Stdin)
	}
	if err := e.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	pattern, err := engine.Select(e)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	written, err := render.Generate(pattern, e, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("Matched pattern: %s\n", pattern.Title)
	fmt.Printf("Wrote %d files to %s:\n", len(written), *out)
	for _, w := range written {
		fmt.Println("  -", w)
	}
	fmt.Println("\nNext: review runbook.md, then save your break-glass kubeconfig BEFORE applying.")
}

func cmdList() {
	fmt.Println("Known auth patterns:")
	for _, p := range catalog.Patterns {
		fmt.Printf("  %-26s %s (OCP %s+, %s)\n", p.ID, p.Title, p.MinOCPVersion, p.OCPIntegration)
	}
}
