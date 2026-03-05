# ghnote

![CI](https://img.shields.io/github/actions/workflow/status/Peter33333/github-note/ci.yml) ![GitHub Release](https://img.shields.io/github/v/release/Peter33333/github-note?display_name=tag) ![GitHub Release Date](https://img.shields.io/github/release-date/Peter33333/github-note?display_date=created_at)

A Linux TUI tool for browsing GitHub issue parent/sub tree and opening issue pages.

## Motivation

I use GitHub Issues as my personal note system.
On the GitHub web UI, issues are mostly presented in a flat list, which makes parent/sub issue relationships hard to read.
This tool exists to present issue hierarchy in a clear tree view, so note structure is visible at a glance.

## Features

- Interactive Personal Access Token login (no OAuth App required)
- Support `GH_TOKEN` environment variable
- Optional OAuth Device Flow fallback (when `client_id` is configured)
- Load issue tree from a single repository
- TUI list view with tree indentation
- Open selected issue in system browser

## UI Preview

![ghnote UI screenshot](https://upload.cc/i1/2026/03/05/tvoMja.png)

## Quick Start

1. Install via APT:

```bash
curl -fsSL https://Peter33333.github.io/github-note/gpg.key | \
sudo gpg --dearmor -o /usr/share/keyrings/ghnote-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/ghnote-keyring.gpg] https://Peter33333.github.io/github-note stable main" | \
sudo tee /etc/apt/sources.list.d/ghnote.list

sudo apt update
sudo apt install ghnote
```

Or install by downloading the binary directly:

```bash
VERSION="$(curl -fsSL https://api.github.com/repos/Peter33333/github-note/releases/latest | grep -o '"tag_name": "[^"]*' | cut -d'"' -f4 | sed 's/^v//')"
ARCH="amd64" # change to arm64 if needed

curl -fsSL -o ghnote.tar.gz \
	"https://github.com/Peter33333/github-note/releases/download/v${VERSION}/ghnote_${VERSION}_linux_${ARCH}.tar.gz"

tar -xzf ghnote.tar.gz
sudo install -m 0755 ghnote /usr/local/bin/ghnote
```

2. Create config template:

```bash
ghnote --init-config
```

3. Edit config file at `~/.config/ghnote/config.yaml`:

```yaml
base_url: https://api.github.com
owner: your_owner
repo: your_repo
```

Optional (only if you want OAuth Device Flow fallback):

```yaml
client_id: your_github_oauth_client_id
```

4. Run:

```bash
ghnote
```

If your config is inside project folder:

```bash
ghnote --config ./configs/config.yaml
```

### Basic Usage Tips

- First login: paste a GitHub Personal Access Token when prompted.
- Navigation: use `j`/`k` (or arrow keys) to move the cursor.
- Tree control: use `h`/`left` to collapse, `l`/`right` to expand, `space` to toggle.
- Open issue: press `enter` to open the selected issue in your browser.
- Fast retry: when auth or repo config changes, rerun `ghnote` directly.

## Key Bindings

- `j` / `down`: move cursor down
- `k` / `up`: move cursor up
- `h` / `left`: collapse current node
- `l` / `right`: expand current node
- `space`: toggle collapse/expand
- `enter`: open selected issue URL
- `q`: quit

## Project Structure

```
cmd/ghnote/             # CLI entry
internal/app/           # startup and orchestration
internal/config/        # config and token storage
internal/domain/        # issue tree domain model
internal/github/        # github auth + issue API
internal/open/          # browser opener
internal/tui/           # bubbletea TUI
configs/                # example config
```

## Notes

- The app currently targets Linux (`xdg-open` for URL launch).
- OAuth client id is optional.
- Recommended login: paste a GitHub Personal Access Token when prompted.
- Token is stored at `~/.config/ghnote/token.yaml`.

## Install (APT)

Add GitHub Pages as an APT source:

```bash
curl -fsSL https://Peter33333.github.io/github-note/gpg.key | \
sudo gpg --dearmor -o /usr/share/keyrings/ghnote-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/ghnote-keyring.gpg] https://Peter33333.github.io/github-note stable main" | \
sudo tee /etc/apt/sources.list.d/ghnote.list

sudo apt update
sudo apt install ghnote
```

## Release Pipeline

- CI: `.github/workflows/ci.yml` runs `go test ./...` on push/PR.
- Release: `.github/workflows/release.yml` runs on tags like `v1.0.0`.
- Packaging: `.goreleaser.yaml` builds Linux binaries and `.deb` packages.
- APT repo publishing: `morph027/apt-repo-action@v3` creates a signed apt repository and deploys via GitHub Pages.

Required GitHub Secrets:

- `APT_SIGNING_KEY`: ASCII armored private GPG key
- `APT_SIGNING_KEY_PASSPHRASE`: passphrase for the private key

To release a new version:

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Star History

[![Star History Chart](https://api.star-history.com/image?repos=Peter33333/github-note&type=timeline&logscale&legend=top-left)](https://www.star-history.com/?repos=Peter33333%2Fgithub-note&type=timeline&legend=top-left)