package cli

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fuse/pmx/internal/config"
	"github.com/fuse/pmx/internal/openid"
	"github.com/fuse/pmx/internal/session"
)

func newAuthCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate against Proxmox (OpenID SSO)",
	}
	cmd.AddCommand(newAuthLoginCmd(opts))
	cmd.AddCommand(newAuthLogoutCmd(opts))
	cmd.AddCommand(newAuthStatusCmd(opts))
	cmd.AddCommand(newAuthPrintEnvCmd(opts))
	return cmd
}

func newAuthLoginCmd(opts *options) *cobra.Command {
	var (
		endpoint     string
		realm        string
		redirectURL  string
		code         string
		state        string
		noBrowser    bool
		sessionPath  string
		callbackPort int
		timeout      time.Duration
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "OpenID login and obtain a Proxmox API ticket",
		Long: `Authenticate with a Proxmox OpenID realm (SSO), then exchange the
authorization code for a Proxmox session ticket and CSRF token.

Flow:
  1. Listen on http://127.0.0.1:<port>/callback
  2. Open the IdP login in a browser
  3. Capture code/state when the browser redirects back
  4. Exchange them for a Proxmox ticket

The callback URL must be registered on the identity provider (Entra / Azure AD).

Create a starter config with: pmx config init`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgFile, err := opts.loadConfig()
			if err != nil {
				return err
			}
			cfg, err := config.ResolveAuth(cfgFile, endpoint, realm, redirectURL, sessionPath, callbackPort)
			if err != nil {
				return err
			}

			client := &openid.Client{Endpoint: cfg.Endpoint}
			ctx := cmd.Context()

			if code == "" || state == "" {
				code, state, cfg.RedirectURL, err = runCallbackLogin(cmd, client, cfg, noBrowser, timeout)
				if err != nil {
					return err
				}
			}

			sess, err := client.Login(ctx, code, state, cfg.RedirectURL)
			if err != nil {
				return err
			}

			ttl, err := cfgFile.SessionTTLDuration()
			if err != nil {
				return err
			}

			if err := session.Save(cfg.SessionFile, cfg.Endpoint, cfg.Realm, sess, ttl); err != nil {
				return fmt.Errorf("save session: %w", err)
			}

			path := cfg.SessionFile
			if path == "" {
				path, _ = session.DefaultPath()
			}

			stored, err := session.Load(path)
			if err != nil {
				return fmt.Errorf("load saved session: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Logged in as %s\n", sess.Username)
			fmt.Fprintf(cmd.OutOrStdout(), "Session written to %s (mode 0600)\n", path)
			fmt.Fprintf(cmd.OutOrStdout(), "Expires at %s\n", stored.ExpiresAt.Format("2006-01-02 15:04:05 UTC"))
			return nil
		},
	}

	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Proxmox base URL (overrides config)")
	cmd.Flags().StringVar(&realm, "realm", "", "OpenID realm id (overrides config)")
	cmd.Flags().StringVar(&redirectURL, "redirect-url", "", "OAuth redirect URL (overrides config)")
	cmd.Flags().StringVar(&code, "code", "", "Authorization code (skip browser flow)")
	cmd.Flags().StringVar(&state, "state", "", "OAuth state (use with --code)")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open a browser automatically")
	cmd.Flags().IntVar(&callbackPort, "callback-port", 0, "Loopback port for callback login (default 8765)")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "How long to wait for the browser callback")
	cmd.Flags().StringVar(&sessionPath, "session-file", "", "Path to write the session JSON (overrides config)")

	return cmd
}

func runCallbackLogin(cmd *cobra.Command, client *openid.Client, cfg config.File, noBrowser bool, timeout time.Duration) (code, state, redirectURL string, err error) {
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()

	redirectURL = cfg.RedirectURL
	actualRedirect, wait, cleanup, err := openid.ListenRedirect(ctx, redirectURL)
	if err != nil {
		return "", "", "", err
	}
	defer func() { _ = cleanup() }()
	redirectURL = actualRedirect

	authURL, err := client.AuthURL(ctx, cfg.Realm, redirectURL)
	if err != nil {
		return "", "", "", err
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "Listening for IdP redirect on:")
	fmt.Fprintln(cmd.ErrOrStderr(), " ", redirectURL)
	fmt.Fprintln(cmd.ErrOrStderr())
	fmt.Fprintln(cmd.ErrOrStderr(), "Open this URL to authenticate:")
	fmt.Fprintln(cmd.ErrOrStderr(), authURL)
	fmt.Fprintln(cmd.ErrOrStderr())
	fmt.Fprintln(cmd.ErrOrStderr(), "Waiting for browser callback…")

	if !noBrowser {
		if err := openBrowser(authURL); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Could not open browser automatically: %v\n", err)
		}
	}

	res, err := wait()
	if err != nil {
		return "", "", "", fmt.Errorf("wait for callback: %w", err)
	}
	return res.Code, res.State, redirectURL, nil
}

func newAuthLogoutCmd(opts *options) *cobra.Command {
	var sessionPath string

	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved session file",
		Long:  `Delete the local session JSON (ticket and CSRF token). Idempotent if no file exists.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := opts.resolveSessionPath(sessionPath)
			if err != nil {
				return err
			}
			if path == "" {
				path, err = session.DefaultPath()
				if err != nil {
					return err
				}
			}

			removed, err := session.Remove(path)
			if err != nil {
				return fmt.Errorf("remove session: %w", err)
			}
			if removed {
				fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", path)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No saved session")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionPath, "session-file", "", "Path to the session JSON (overrides config)")
	return cmd
}

func newAuthStatusCmd(opts *options) *cobra.Command {
	var (
		sessionPath string
		checkAPI    bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the saved session and expiry",
		Long: `Show the saved session file, estimated expiry, and whether the ticket
looks expired locally. Use --check to call the Proxmox API (/version).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := opts.resolveSessionPath(sessionPath)
			if err != nil {
				return err
			}
			stored, err := session.Load(path)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			expired := stored.ExpiredAt(now)
			remaining := stored.Remaining(now)

			fmt.Fprintf(cmd.OutOrStdout(), "username: %s\n", stored.Username)
			fmt.Fprintf(cmd.OutOrStdout(), "endpoint: %s\n", stored.Endpoint)
			fmt.Fprintf(cmd.OutOrStdout(), "realm:    %s\n", stored.Realm)
			fmt.Fprintf(cmd.OutOrStdout(), "obtained: %s\n", stored.ObtainedAt.Format("2006-01-02 15:04:05 UTC"))
			fmt.Fprintf(cmd.OutOrStdout(), "expires:  %s\n", stored.ExpiresAt.Format("2006-01-02 15:04:05 UTC"))
			fmt.Fprintf(cmd.OutOrStdout(), "remaining: %s\n", formatRemainingMinutes(remaining))
			if expired {
				fmt.Fprintln(cmd.OutOrStdout(), "expired:  yes (local estimate)")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "expired:  no")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ticket:   %s…\n", truncate(stored.Ticket, 24))

			if checkAPI {
				client := &openid.Client{Endpoint: stored.Endpoint}
				if err := client.VerifyTicket(cmd.Context(), stored.Ticket); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "api:      rejected (%v)\n", err)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "api:      accepted")
				}
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&sessionPath, "session-file", "", "Path to the session JSON (overrides config)")
	cmd.Flags().BoolVar(&checkAPI, "check", false, "Verify the ticket against the Proxmox API")
	return cmd
}

func newAuthPrintEnvCmd(opts *options) *cobra.Command {
	var sessionPath string

	cmd := &cobra.Command{
		Use:   "print-env",
		Short: "Print shell exports for the saved ticket (for curl / Terraform env)",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := opts.resolveSessionPath(sessionPath)
			if err != nil {
				return err
			}
			stored, err := session.Load(path)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "export PMX_ENDPOINT=%q\n", stored.Endpoint)
			fmt.Fprintf(cmd.OutOrStdout(), "export PMX_TICKET=%q\n", stored.Ticket)
			fmt.Fprintf(cmd.OutOrStdout(), "export PMX_CSRF=%q\n", stored.CSRFPreventionToken)
			fmt.Fprintf(cmd.OutOrStdout(), "export PROXMOX_VE_ENDPOINT=%q\n", strings.TrimRight(stored.Endpoint, "/")+"/api2/json")
			fmt.Fprintf(cmd.OutOrStdout(), "export PROXMOX_VE_AUTH_TICKET=%q\n", stored.Ticket)
			fmt.Fprintf(cmd.OutOrStdout(), "export PROXMOX_VE_CSRF_PREVENTION_TOKEN=%q\n", stored.CSRFPreventionToken)
			return nil
		},
	}
	cmd.Flags().StringVar(&sessionPath, "session-file", "", "Path to the session JSON (overrides config)")
	return cmd
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported OS %q", runtime.GOOS)
	}
	return cmd.Start()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func formatRemainingMinutes(d time.Duration) string {
	if d <= 0 {
		return "0 min"
	}
	mins := int((d + time.Minute - 1) / time.Minute)
	return fmt.Sprintf("%d min", mins)
}
