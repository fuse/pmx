package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fuse/pmx/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage local YAML configuration",
	}
	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigInitCmd())
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the default config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func newConfigInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a starter config.yaml (generic placeholders)",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultPath()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				return err
			}
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", path)
			}
			content := `# Proxmox OpenID settings for pmx.

endpoint: https://pve.example.com
realm: my-openid-realm

# Local callback captures the IdP redirect automatically.
# Register http://127.0.0.1:8765/callback on your identity provider.
callback_port: 8765
`
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", path)
			fmt.Fprintln(cmd.OutOrStdout(), "Edit endpoint and realm, then run: pmx auth login")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing config file")
	return cmd
}
