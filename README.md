# ghnote

A Linux TUI tool for browsing GitHub issue parent/sub tree and opening issue pages.

## Features

- OAuth Device Flow login for GitHub
- Load issue tree from a single repository
- TUI list view with tree indentation
- Open selected issue in system browser

## Quick Start

1. Build:

```bash
go build -o ghnote ./cmd/ghnote
```

2. Create config template:

```bash
./ghnote --init-config
```

3. Edit config file at `~/.config/ghnote/config.yaml`:

```yaml
client_id: your_github_oauth_client_id
base_url: https://api.github.com
owner: your_owner
repo: your_repo
```

4. Run:

```bash
./ghnote
```

If your config is inside project folder:

```bash
./ghnote --config ./configs/config.yaml
```

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
- OAuth client id is required.
- Token is stored at `~/.config/ghnote/token.yaml`.
