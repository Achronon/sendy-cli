# sendy — command-line client for sendy.md

Single static binary. Go stdlib + `github.com/zalando/go-keyring`.

> **Source-of-truth repo.** The server, web app, and macOS app live in
> the private [Achronon/sendy-md](https://github.com/Achronon/sendy-md)
> monorepo. This repo holds just the CLI so binaries can be published
> on public GitHub Releases for `brew install` and curl-based install.

## Install

### Homebrew (macOS, Linux)

```bash
brew install achronon/sendy-md/sendy
```

The tap lives at
[Achronon/homebrew-sendy-md](https://github.com/Achronon/homebrew-sendy-md)
and is bumped automatically on every `v*` tag here.

### Pre-built binaries

```bash
# macOS (Apple silicon)
curl -fsSL https://github.com/Achronon/sendy-cli/releases/latest/download/sendy-darwin-arm64 -o /usr/local/bin/sendy
chmod +x /usr/local/bin/sendy

# macOS (Intel)
curl -fsSL https://github.com/Achronon/sendy-cli/releases/latest/download/sendy-darwin-amd64 -o /usr/local/bin/sendy

# Linux (amd64)
curl -fsSL https://github.com/Achronon/sendy-cli/releases/latest/download/sendy-linux-amd64 -o /usr/local/bin/sendy
```

Verify with `sendy version`.

### From source

```bash
go install github.com/Achronon/sendy-cli@latest
```

## Usage

```
sendy [FILE|-]              Create paste from file, stdin, or `-`
sendy create [FILE|-]       Explicit create
sendy list [--limit N]      List your pastes (--search Q to filter by content)
sendy view <slug>           Print a paste's content
sendy raw <slug>            Print raw text of a paste
sendy login                 Sign in via the browser (session token → OS keyring, PKCE S256)
sendy logout                Remove the stored token
sendy whoami                Show configured identity
sendy claim [--user-key K]  Migrate anonymous pastes to the signed-in account
sendy completions SHELL     Print shell-completion script (bash | zsh | fish)
```

### Environment

| Var                   | Default             | Purpose                                 |
|-----------------------|---------------------|-----------------------------------------|
| `SENDY_URL`           | `https://sendy.md`  | API base                                |
| `SENDY_USER_KEY`      | `(unset)`           | Anonymous identifier (before `login`)   |
| `SENDY_SESSION_TOKEN` | `(unset)`           | Overrides the keyring; useful in CI     |

### Shell completions

```bash
# bash
sendy completions bash | sudo tee /etc/bash_completion.d/sendy

# zsh — directory must be on $fpath, e.g. ~/.zsh/completions
mkdir -p ~/.zsh/completions
sendy completions zsh > ~/.zsh/completions/_sendy

# fish
sendy completions fish > ~/.config/fish/completions/sendy.fish
```

## Examples

```bash
echo "hello" | sendy
sendy README.md
sendy login && sendy list --limit 10
sendy view abc123 --password hunter2
sendy list --search "certificate"
sendy claim --user-key oldAnonKey
```

## Security

`sendy login` uses the RFC 7636 PKCE (S256) flow — a captured `code`
from the browser callback is useless without the in-memory verifier
held by the CLI that initiated the flow.

## Development

```bash
go build -o sendy .    # local binary
go test ./...          # unit tests
go vet ./...           # static analysis
```

## Cutting a release

```bash
git tag v0.1.2
git push origin v0.1.2
```

`.github/workflows/release.yml` picks it up, cross-compiles for
darwin/linux × amd64/arm64, attaches the binaries + `SHA256SUMS` to a
GitHub Release, and auto-bumps the Homebrew formula (requires the
`HOMEBREW_TAP_TOKEN` secret with contents:rw on the tap repo).

## License

MIT.
