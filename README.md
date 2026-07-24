# tpack - A Tmux Plugin Manager

[![CI](https://github.com/tmuxpack/tpack/actions/workflows/ci.yml/badge.svg)](https://github.com/tmuxpack/tpack/actions/workflows/ci.yml)
[![Release](https://github.com/tmuxpack/tpack/actions/workflows/release.yml/badge.svg)](https://github.com/tmuxpack/tpack/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/tmuxpack/tpack)](https://github.com/tmuxpack/tpack/releases/latest)
[![AUR](https://img.shields.io/aur/version/tpack-bin)](https://aur.archlinux.org/packages/tpack-bin)
[![Homebrew](https://img.shields.io/badge/homebrew-tap-orange)](https://github.com/tmuxpack/homebrew-tpack)
[![Go Version](https://img.shields.io/github/go-mod/go-version/tmuxpack/tpack)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE.md)

<img width="718" height="489" alt="image" src="https://github.com/user-attachments/assets/62b32a4d-5320-4198-a6a9-248441e89f94" />
<img width="719" height="489" alt="image" src="https://github.com/user-attachments/assets/42872613-3348-4979-b463-736895c15911" />


> **[Full Documentation](https://tmuxpack.github.io/tpack/)** - Mostly AI-generated

A modern tmux plugin manager written in Go. It accepts existing
[TPM](https://github.com/tmux-plugins/tpm) configuration syntax and key
bindings, while adding repository-aware plugin management.

> [!IMPORTANT]
> tpack stores plugins in repository-specific directories such as
> `tmux-87a1216f1f68` (always ASCII and at most 64 bytes), so repositories with
> the same basename can coexist. On the first operational command, tpack
> identifies a legacy basename checkout by its Git origin and renames an exact
> match once. A mismatched or unreadable checkout is never guessed, deleted, or
> moved, and concurrent tpack runs cannot overwrite a canonical directory.
> Explicit `alias=` values keep their configured directory instead.
>
> This means switching back to TPM is not seamless. Running tpack and TPM
> against the same plugin root is unsupported, and scripts or configuration
> with hard-coded plugin paths may need updating.

Works on Linux, macOS, and FreeBSD.

## Installation

Requirements: `tmux` version 1.9 (or higher), `git`, `bash`.

There are two ways to set up tpack, each with different installation methods and
configuration. Pick whichever fits your workflow.

### Option A: Standalone binary

Install tpack as a binary on your `$PATH`. This is the recommended approach for
new setups.

<details>
<summary><b>Homebrew (macOS / Linux)</b></summary>

Custom tap repository (updated immediately after new release):

```bash
brew install tmuxpack/tpack/tpack
```

or official formula (updated less frequently):

```bash
brew install tpack
```

</details>

<details>
<summary><b>AUR (Arch Linux)</b></summary>

```bash
yay -S tpack-bin
```

</details>

<details>
<summary><b>DEB / RPM (Debian, Ubuntu, Fedora, etc.)</b></summary>

Download the latest `.deb` or `.rpm` package from the
[releases page](https://github.com/tmuxpack/tpack/releases/latest) and install
it with your package manager:

```bash
# Debian / Ubuntu
sudo dpkg -i tpack_*.deb

# Fedora / RHEL
sudo rpm -i tpack_*.rpm
```

</details>

<details>
<summary><b>Go install</b></summary>

Requires [Go](https://go.dev/) 1.26 or later:

```bash
go install github.com/tmuxpack/tpack/cmd/tpack@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is on your `$PATH`.

</details>

<details>
<summary><b>Build from source</b></summary>

```bash
git clone https://github.com/tmuxpack/tpack
cd tpack
make build
# Binary is at dist/tpack — move it somewhere on your $PATH
sudo cp dist/tpack /usr/local/bin/
```

</details>

Then add to `~/.tmux.conf` (or `$XDG_CONFIG_HOME/tmux/tmux.conf`):

```bash
# List of plugins
set -g @plugin 'tmux-plugins/tmux-sensible'

# Initialize tpack (keep this line at the very bottom of tmux.conf)
run 'tpack init'
```

### Option B: Git clone (TPM-compatible setup)

Clone tpack into the TPM directory. Existing TPM declaration syntax remains
valid, so no `tmux.conf` syntax changes are needed when switching from TPM. The
plugin directory migration described above still applies.

```bash
git clone https://github.com/tmuxpack/tpack ~/.tmux/plugins/tpm
```

Then add to `~/.tmux.conf` (or `$XDG_CONFIG_HOME/tmux/tmux.conf`):

```bash
# List of plugins
set -g @plugin 'tmux-plugins/tmux-sensible'

# Initialize tpack (keep this line at the very bottom of tmux.conf)
run '~/.tmux/plugins/tpm/tpm'
```

### Load and install plugins

Reload tmux and press `prefix` + <kbd>I</kbd> to install plugins:

```bash
tmux source ~/.tmux.conf
```

See the [Getting Started](https://tmuxpack.github.io/tpack/getting-started/)
guide for full setup instructions.

## Features

- **TPM-compatible configuration** — existing declarations and key bindings work
- **Interactive TUI** — browse, install, update, remove, and uninstall plugins visually
  (`prefix` + <kbd>T</kbd>)
- **CLI** — `tpack install`, `tpack update`, `tpack clean`, and more
- **Automatic updates** — optional background update checking for plugins and
  tpack itself
- **Customizable** — key bindings, colors, plugin directory, and update behavior
- **Plugins browser** — search, browse and install plugins from the TUI, the list being maintained on the [plugins-registry](https://github.com/tmuxpack/plugins-registry)

Not interested in a whole category of plugins (e.g. `ai`)? Hide it from the
browser with `@tpack-hidden-categories`:

```bash
set -g @tpack-hidden-categories 'ai'
```

See the [full documentation](https://tmuxpack.github.io/tpack/) for details on
configuration, usage, and the CLI reference.

## Migrating from TPM

tpack accepts TPM configuration syntax. Two ways to switch:

- **Git remote** — if you `git clone`d TPM, just point the remote at tpack and
  pull. No `tmux.conf` changes needed:

  ```bash
  cd ~/.tmux/plugins/tpm
  git remote set-url origin https://github.com/tmuxpack/tpack
  git pull
  ```

- **Package manager** — install tpack via Homebrew, AUR, DEB/RPM, or
  `go install` (see [installation guide](https://tmuxpack.github.io/tpack/getting-started/installation/)),
  then replace the `run` line in your `tmux.conf` with `run 'tpack init'`.

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
