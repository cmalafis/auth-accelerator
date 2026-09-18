// Command auth-accelerator generates the YAML and runbook to enable smart-card
// authentication on an OpenShift 4.20+ cluster, based on the components the admin
// has available.
//
// The CLI is built with spf13/cobra; commands live in the cmd package. Remaining
// upgrade seams (see CLAUDE.md):
//   - swap env.Prompt for charmbracelet/huh (the "select what you have" wizard)
//   - add a --from-cluster path using k8s.io/client-go to introspect the cluster
package main

import "github.com/cmalafis/auth-accelerator/cmd"

func main() {
	cmd.Execute()
}
