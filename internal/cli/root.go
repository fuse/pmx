package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:           "pmx",
		Short:         "Proxmox OpenID auth helper",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "Path to YAML config (default: ~/.config/pmx/config.yaml)")
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newAuthCmd(opts))
	cmd.AddCommand(newConfigCmd())
	return cmd
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
