# tpack - Tmux Plugin Manager

[![CI](https://github.com/tmuxpack/tpack/actions/workflows/ci.yml/badge.svg)](https://github.com/tmuxpack/tpack/actions/workflows/ci.yml)
[![Release](https://github.com/tmuxpack/tpack/actions/workflows/release.yml/badge.svg)](https://github.com/tmuxpack/tpack/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/tmuxpack/tpack)](https://github.com/tmuxpack/tpack/releases/latest)
[![AUR](https://img.shields.io/aur/version/tpack-bin)](https://aur.archlinux.org/packages/tpack-bin)
[![Homebrew](https://img.shields.io/homebrew/cask/v/tpack)](https://github.com/tmuxpack/homebrew-tpack)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tmuxpack/tpack)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.md)

A modern tmux plugin manager written in Go. **Drop-in replacement for
[TPM](https://github.com/tmux-plugins/tpm)** — fully backward compatible with
existing TPM configurations, plugins, and key bindings.

Works on Linux, macOS, and FreeBSD.

See list of plugins [here](https://github.com/tmux-plugins/list).

## Installation

Requirements: `tmux` version 1.9 (or higher), `git`, `bash`.

### Homebrew (macOS / Linux)

```bash
brew install --cask tmuxpack/tpack/tpack
```

### AUR (Arch Linux)

```bash
yay -S tpack-bin
```

### DEB / RPM (Debian, Ubuntu, Fedora, etc.)

Download the latest `.deb` or `.rpm` package from the
[releases page](https://github.com/tmuxpack/tpack/releases/latest) and install
it with your package manager:

```bash
# Debian / Ubuntu
sudo dpkg -i tpack_*.deb

# Fedora / RHEL
sudo rpm -i tpack_*.rpm
```

### Git clone

```bash
git clone https://github.com/tmuxpack/tpack ~/.tmux/plugins/tpm
```

### Build from source

```bash
git clone https://github.com/tmuxpack/tpack
cd tpack
make build
# Binary is at dist/tpack
```

### Configuration

Add to `~/.tmux.conf` (or `$XDG_CONFIG_HOME/tmux/tmux.conf`):

```bash
# List of plugins
set -g @plugin 'tmuxpack/tpack'
set -g @plugin 'tmux-plugins/tmux-sensible'

# Other examples:
# set -g @plugin 'github_username/plugin_name'
# set -g @plugin 'github_username/plugin_name#branch'
# set -g @plugin 'git@github.com:user/plugin'
# set -g @plugin 'git@bitbucket.com:user/plugin'
# set -g @plugin 'user/plugin alias=custom_name'

# Initialize tpack (keep this line at the very bottom of tmux.conf)
run '~/.tmux/plugins/tpm/tpm'
```

Reload tmux so tpack is sourced:

```bash
tmux source ~/.tmux.conf
```

## TPM compatibility

tpack is a **drop-in replacement** for TPM. You can switch without changing your
`tmux.conf` — all existing TPM settings, plugin declarations, environment
variables, and key bindings continue to work:

- `set -g @tpm_plugins '...'` (legacy list syntax) and `set -g @plugin '...'`
  are both supported
- `@tpm-install`, `@tpm-update`, `@tpm-clean` options still work alongside the
  `@tpack-*` equivalents
- `TMUX_PLUGIN_MANAGER_PATH` is honored (as well as `TPACK_PLUGIN_PATH`)
- Installs to `~/.tmux/plugins/tpm` — the same path TPM uses

## Installing plugins

1. Add new plugin to `~/.tmux.conf` with `set -g @plugin '...'`
2. Press `prefix` + <kbd>I</kbd> (capital i, as in **I**nstall) to fetch the plugin.

The plugin is cloned to `~/.tmux/plugins/` and sourced automatically.

## Uninstalling plugins

1. Remove (or comment out) plugin from the list.
2. Press `prefix` + <kbd>alt</kbd> + <kbd>u</kbd> (lowercase u as in **u**ninstall) to remove the plugin.

All plugins are installed to `~/.tmux/plugins/` so alternatively you can
find the plugin directory there and remove it.

## Interactive TUI

tpack includes a built-in terminal UI for managing plugins interactively.

Press `prefix` + <kbd>T</kbd> to open the TUI (launches in a tmux popup on
tmux 3.2+, falls back to inline on older versions).

From the TUI you can browse installed plugins, install, update, uninstall, or
clean with multi-select, watch progress in real time, and inspect commit
history for recent updates.

The install, update, and clean key bindings (`prefix` + <kbd>I</kbd>,
`prefix` + <kbd>U</kbd>, `prefix` + <kbd>alt</kbd> + <kbd>u</kbd>) also open
the TUI with the corresponding operation pre-selected.

## CLI

The `tpack` binary can be used directly from the command line:

```bash
tpack install          # Install all plugins from tmux.conf
tpack update [name]    # Update a specific plugin (or all)
tpack clean            # Remove plugins not in tmux.conf
tpack source           # Source plugins without installing
tpack tui              # Open the interactive TUI
tpack version          # Print version
```

See also [Managing plugins via the command line](docs/managing_plugins_via_cmd_line.md)
for the shell wrapper scripts.

## Key bindings

`prefix` + <kbd>I</kbd>
- Installs new plugins from GitHub or any other git repository
- Refreshes tmux environment

`prefix` + <kbd>U</kbd>
- Updates plugin(s)

`prefix` + <kbd>alt</kbd> + <kbd>u</kbd>
- Remove/uninstall plugins not on the plugin list

`prefix` + <kbd>T</kbd>
- Opens the interactive TUI

All key bindings can be customized via tmux options:

```bash
set -g @tpack-install 'I'   # default: I
set -g @tpack-update  'U'   # default: U
set -g @tpack-clean   'M-u' # default: M-u
set -g @tpack-tui     'T'   # default: T
```

## Automatic updates

tpack can periodically check for plugin updates in the background. To enable,
add to `tmux.conf`:

```bash
set -g @tpack-update-mode 'prompt'       # "prompt", "auto", or "off" (default: off)
set -g @tpack-update-interval '24h'      # how often to check (Go duration)
```

- **prompt** — display a message when updates are available; you update manually.
- **auto** — automatically update outdated plugins in the background.
- **off** — disable update checking (default).

### Self-update

When tpack is installed via auto-download or git clone, the binary can update
itself from GitHub releases. It checks once every 24 hours. To pin a specific
version and disable self-update:

```bash
set -g @tpack-version '1.2.3'
```

## Color customization

Override the TUI color palette with tmux options:

```bash
set -g @tpack-color-primary   '#89b4fa'
set -g @tpack-color-secondary '#a6e3a1'
set -g @tpack-color-accent    '#f9e2af'
set -g @tpack-color-error     '#f38ba8'
set -g @tpack-color-muted     '#6c7086'
set -g @tpack-color-text      '#cdd6f4'
```

## Docs

- [Help, tpack not working](docs/tpack_not_working.md) - problem solutions

More advanced features and instructions, regular users probably do not need
this:

- [How to create a plugin](docs/how_to_create_plugin.md). It's easy.
- [Managing plugins via the command line](docs/managing_plugins_via_cmd_line.md)
- [Changing plugins install dir](docs/changing_plugins_install_dir.md)
- [Automatic tpack installation on a new machine](docs/automatic_tpack_installation.md)

## Tests

Run tests with:

```bash
make test
```

## Acknowledgments

tpack is built on the foundations of [TPM](https://github.com/tmux-plugins/tpm),
the original Tmux Plugin Manager created by
[Bruno Sutic](https://github.com/bruno-). Thanks to Bruno and the TPM
contributors for establishing the plugin ecosystem that tpack is designed to be
compatible with.

## License

[MIT](LICENSE.md)
