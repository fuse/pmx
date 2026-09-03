package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "pmx %s\n", Version)
			fmt.Fprintf(cmd.OutOrStdout(), "commit: %s\n", Commit)
			fmt.Fprintf(cmd.OutOrStdout(), "built:  %s\n", BuildTime)
			return nil
		},
	}
}
