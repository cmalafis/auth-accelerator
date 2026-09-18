package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cmalafis/auth-accelerator/internal/catalog"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the known auth patterns",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Known auth patterns:")
			for _, p := range catalog.Patterns {
				fmt.Printf("  %-26s %s (OCP %s+, %s)\n", p.ID, p.Title, p.MinOCPVersion, p.OCPIntegration)
			}
		},
	}
}
