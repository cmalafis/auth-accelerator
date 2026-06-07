// Package cmd wires the auth-accelerator subcommands together with cobra.
//
// This is the CLI layer only. The value lives in the internal packages:
//   - internal/env      the Environment the admin declares
//   - internal/catalog  the AuthPattern knowledge base (the IP)
//   - internal/engine   the deterministic Select(env) -> *AuthPattern
//   - internal/render   go:embed templates -> artifacts
//
// cobra owns nothing but flag parsing, help text, and dispatch.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "auth-accelerator",
	Short: "smart-card auth setup for OpenShift 4.20+",
	Long: "auth-accelerator — smart-card auth setup for OpenShift 4.20+\n\n" +
		"Generate the Authentication CR, an IdP recipe, a runbook, and an ADR to enable\n" +
		"smart-card (CAC/PIV) authentication on an OpenShift 4.20+ cluster.",
	Version: version,
	// RunE errors are printed once by Execute in our own "error: <msg>" style.
	SilenceUsage:  true,
	SilenceErrors: true,
	// No subcommand: print usage to stderr and exit non-zero, matching the
	// pre-cobra behavior (usage() + os.Exit(2)).
	Run: func(cmd *cobra.Command, args []string) {
		cmd.SetOut(os.Stderr)
		_ = cmd.Help()
		os.Exit(2)
	},
}

func init() {
	rootCmd.AddCommand(newGenerateCmd(), newListCmd(), newVersionCmd())
}

// Execute runs the root command and maps failures to the legacy exit behavior:
// an "error: <msg>" line on stderr and a non-zero exit code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
