---
hide:
  - navigation
  - toc
---

# tpack

**A modern tmux plugin manager written in Go.**
Drop-in replacement for TPM — fully backward compatible with existing configurations, plugins, and key bindings.

Works on Linux, macOS, and FreeBSD.

---

## Features

<div class="grid cards" markdown>

-   **Drop-in TPM Replacement**

    Switch without changing your `tmux.conf`. All existing TPM settings, plugin declarations, and key bindings continue to work.

-   **Interactive TUI**

    Browse, install, update, and clean plugins with a built-in terminal UI. Launches in a tmux popup on 3.2+.

-   **Automatic Updates**

    Background plugin update checking with three modes: prompt, auto, or off. Self-update support for the tpack binary itself.

-   **Multiple Install Methods**

    Available via Homebrew, AUR, deb/rpm packages, git clone, or build from source.

</div>

## Quick Start

Install tpack:

```bash
# Install (macOS / Linux)
brew install tmuxpack/tpack/tpack
```

Add plugins to your tmux configuration:

```bash
# ~/.tmux.conf
set -g @plugin 'tmuxpack/tpack'
set -g @plugin 'tmux-plugins/tmux-sensible'

# Initialize tpack (keep this line at the very bottom of tmux.conf)
run '~/.tmux/plugins/tpm/tpm'
```

Reload tmux:

```bash
tmux source ~/.tmux.conf
```

See the [installation guide](getting-started/installation.md) for all install methods.
