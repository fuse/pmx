# pmx

CLI helper for Proxmox OpenID SSO authentication.

Obtain a Proxmox API session ticket bound to your identity-provider user, then use it with `curl` (or Terraform via `PROXMOX_VE_AUTH_TICKET`).

Repository: https://github.com/fuse/pmx

## Install

From a clone:

```bash
git clone https://github.com/fuse/pmx.git
cd pmx
asdf install
go build -o bin/pmx ./cmd/pmx
```

Or install a tagged release binary from [GitHub Releases](https://github.com/fuse/pmx/releases).

Or with `go install` (binary in `$GOPATH/bin` or `$HOME/go/bin`):

```bash
go install github.com/fuse/pmx/cmd/pmx@v0.1.0
```

## Releases

Push a semver tag on `develop` to build release binaries (linux, darwin, windows) with embedded version metadata:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The workflow sets `Version` (tag without `v`), `Commit`, and `BuildTime` via `-ldflags`. Check with:

```bash
pmx version
```

## Configuration

```bash
./bin/pmx config init
# edit ~/.config/pmx/config.yaml
```

Example (`config.example.yaml`):

```yaml
endpoint: https://pve.example.com
realm: my-openid-realm
callback_port: 8765
```

Register this redirect URI on the IdP (Entra / Azure AD app registration):

`http://127.0.0.1:8765/callback`

## Auth

```bash
./bin/pmx auth login
```

The CLI listens on loopback, opens the browser, and captures `code` / `state` when the IdP redirects back.

On success, the ticket is written to `~/.config/pmx/session.json` (mode `0600`). Override with `session_file` in the config or `--session-file`.

Proxmox API tickets expire after a cluster-configured timeout (default **2 hours**). `pmx` stores an estimated `expires_at` based on `session_ttl` in the config (default `2h`). Use `auth status` to see remaining time; add `--check` to verify the ticket against the API.

```bash
./bin/pmx auth status
./bin/pmx auth status --check
```

## Environment variables

`pmx` does **not** keep a `.env` file. `auth print-env` reads the session JSON and prints shell exports for the current process:

| Variable | Source |
| --- | --- |
| `PMX_ENDPOINT`, `PMX_TICKET`, `PMX_CSRF` | session file (for curl) |
| `PROXMOX_VE_ENDPOINT`, `PROXMOX_VE_AUTH_TICKET`, `PROXMOX_VE_CSRF_PREVENTION_TOKEN` | same ticket, Terraform provider naming |

```bash
eval "$(./bin/pmx auth print-env)"
```

Without `eval`, the exports stay in the subshell and disappear. Run `auth status` to check expiry, then `auth login` when needed. Use `auth logout` to delete the local session file.

## Curl check

```bash
eval "$(./bin/pmx auth print-env)"

curl -s -b "PVEAuthCookie=$PMX_TICKET" \
  "$PMX_ENDPOINT/api2/json/version"
```

## Commands

- `pmx version`
- `pmx config init` / `pmx config path`
- `pmx auth login`
- `pmx auth logout`
- `pmx auth status` (`--check` to verify against the API)
- `pmx auth print-env`

## Troubleshooting

**Port already in use** (`listen on 127.0.0.1:8765`)  
Another process holds the callback port. Change `callback_port` in the config or pass `--callback-port`.

**Redirect URI mismatch on the IdP**  
The redirect registered on Entra must match exactly, e.g. `http://127.0.0.1:8765/callback` (not `localhost` if you configured `127.0.0.1`).

**`auth login` TLS / certificate error**  
The Proxmox endpoint must present a certificate trusted by your system. Fix the cert on the cluster or install the issuing CA locally.

**`auth status` shows expired but you just logged in**  
Adjust `session_ttl` in the config if your cluster uses a non-default API ticket lifetime.

**`auth status --check` rejected**  
The ticket is invalid server-side. Run `auth login` again.

**`curl` fails after `print-env`**  
Use `eval "$(./bin/pmx auth print-env)"` in the same shell. Without `eval`, exports stay in a subshell.

## License

MIT. See [LICENSE](LICENSE).
