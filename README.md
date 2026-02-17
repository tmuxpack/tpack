# tpack - A Tmux Plugin Manager

[![CI](https://github.com/tmuxpack/tpack/actions/workflows/ci.yml/badge.svg)](https://github.com/tmuxpack/tpack/actions/workflows/ci.yml)
[![Release](https://github.com/tmuxpack/tpack/actions/workflows/release.yml/badge.svg)](https://github.com/tmuxpack/tpack/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/tmuxpack/tpack)](https://github.com/tmuxpack/tpack/releases/latest)
[![AUR](https://img.shields.io/aur/version/tpack-bin)](https://aur.archlinux.org/packages/tpack-bin)
[![Homebrew](https://img.shields.io/badge/homebrew-tap-orange)](https://github.com/tmuxpack/homebrew-tpack)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tmuxpack/tpack)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.md)

> **[Full Documentation](https://tmuxpack.github.io/tpack/)**

A modern tmux plugin manager written in Go. **Drop-in replacement for
[TPM](https://github.com/tmux-plugins/tpm)** — fully backward compatible with
existing TPM configurations, plugins, and key bindings.

Works on Linux, macOS, and FreeBSD.

## Installation

Requirements: `tmux` version 1.9 (or higher), `git`, `bash`.

### Homebrew (macOS / Linux)

```bash
brew install tmuxpack/tpack/tpack
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

## Quick start

Add to `~/.tmux.conf` (or `$XDG_CONFIG_HOME/tmux/tmux.conf`):

```bash
# List of plugins
set -g @plugin 'tmuxpack/tpack'
set -g @plugin 'tmux-plugins/tmux-sensible'

# Initialize tpack (keep this line at the very bottom of tmux.conf)
run '~/.tmux/plugins/tpm/tpm'
```

Reload tmux and press `prefix` + <kbd>I</kbd> to install plugins:

```bash
tmux source ~/.tmux.conf
```

See the [Getting Started](https://tmuxpack.github.io/tpack/getting-started/)
guide for full setup instructions.

## Features

- **Drop-in TPM replacement** — no config changes needed when switching from TPM
- **Interactive TUI** — browse, install, update, and uninstall plugins visually
  (`prefix` + <kbd>T</kbd>)
- **CLI** — `tpack install`, `tpack update`, `tpack clean`, and more
- **Automatic updates** — optional background update checking for plugins and
  tpack itself
- **Customizable** — key bindings, colors, plugin directory, and update behavior

See the [full documentation](https://tmuxpack.github.io/tpack/) for details on
configuration, usage, and the CLI reference.

## Migrating from TPM

tpack is a drop-in replacement for TPM. Two ways to switch:

- **Git remote** — if you `git clone`d TPM, just point the remote at tpack and
  pull. No `tmux.conf` changes needed:

  ```bash
  cd ~/.tmux/plugins/tpm
  git remote set-url origin https://github.com/tmuxpack/tpack
  git pull
  ```

- **Package manager** — install tpack via Homebrew, AUR, DEB/RPM, or
  `go install`, then replace the `run` line in your `tmux.conf` with
  `run 'tpack init'`.

See the
[full migration guide](https://tmuxpack.github.io/tpack/getting-started/migrating-from-tpm/)
for details.

## Acknowledgments

tpack is built on the foundations of [TPM](https://github.com/tmux-plugins/tpm),
the original Tmux Plugin Manager created by
[Bruno Sutic](https://github.com/bruno-). Thanks to Bruno and the TPM
contributors for establishing the plugin ecosystem that tpack is designed to be
compatible with.

## License

[MIT](LICENSE.md)
