# ghnote

[![CI](https://github.com/Peter33333/github-note/actions/workflows/ci.yml/badge.svg)](https://github.com/Peter33333/github-note/actions/workflows/ci.yml) ![GitHub Release](https://img.shields.io/github/v/release/Peter33333/github-note?display_name=tag) ![GitHub Release Date](https://img.shields.io/github/release-date/Peter33333/github-note?display_date=created_at)

A Linux TUI tool for browsing GitHub issue parent/sub tree and opening issue pages.

## Motivation

I use GitHub Issues as my personal note system.
On the GitHub web UI, issues are mostly presented in a flat list, which makes parent/sub issue relationships hard to read.
This tool exists to present issue hierarchy in a clear tree view, so note structure is visible at a glance.

## Features

- Optional Personal Access Token login (`Enter` to skip for public repositories)
- Support `GH_TOKEN` environment variable
- Lazy loading by page (first page only on startup)
- Load issue tree from a single repository
- TUI list view with tree indentation
- Open selected issue in system browser

## UI Preview

![ghnote UI screenshot](https://upload.cc/i1/2026/03/09/hcPmd2.png)

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

2. First run (interactive setup):

```bash
ghnote
```

If `~/.config/ghnote/config.yaml` does not exist, `ghnote` will start an interactive wizard and ask for:

- `repository` (required), accepts:
	- `https://github.com/ruanyf/weekly/issues`
	- `https://github.com/ruanyf/weekly`
	- `ruanyf/weekly`

Note: the wizard requires an interactive terminal (TTY).

You can also explicitly run the wizard:

```bash
ghnote --init-config
```

3. Run:

```bash
ghnote
```

If your config is inside project folder:

```bash
ghnote --config ./configs/config.yaml
```

### Basic Usage Tips

- First login: paste a GitHub Personal Access Token when prompted, or press `Enter` for public-only mode.
- Navigation: use `j`/`k` (or arrow keys) to move the cursor.
- Page navigation: use `[`/`]` (or `p`/`n`) for previous/next issue page.
- Fast list navigation: use `g`/`G` for first/last item, and `pgup`/`pgdown` for list scroll.
- Tree control: use `h`/`left` to collapse, `l`/`right` to expand, `space` to toggle.
- Open issue: press `enter` to open the selected issue in your browser.
- Help: press `?` to toggle extended help.
- Fast retry: when auth or repo config changes, rerun `ghnote` directly.

## Key Bindings

- `j` / `down`: move cursor down
- `k` / `up`: move cursor up
- `g` / `home`: jump to first item
- `G` / `end`: jump to last item
- `[`, `p`: load previous issue page
- `]`, `n`: load next issue page
- `pgup`: scroll one screen up
- `pgdown`: scroll one screen down
- `h` / `left`: collapse current node
- `l` / `right`: expand current node
- `space`: toggle collapse/expand
- `enter`: open selected issue URL
- `?`: toggle extended help
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
- PAT is optional: without PAT, only public repositories are accessible.
- For private repositories, use a GitHub Personal Access Token with `repo` scope.
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