# Automatic Updates

## Plugin updates

Configure automatic update checking with two tmux options:

```bash
set -g @tpack-update-mode 'prompt'       # "prompt", "auto", or "off" (default: off)
set -g @tpack-update-interval '24h'      # how often to check (Go duration)
```

**Modes:**

- **prompt** — Display a tmux message when updates are available. You decide when to update (e.g. press ++prefix+shift+u++).
- **auto** — Automatically update outdated plugins in the background.
- **off** — Disable update checking (default).

Both `@tpack-update-mode` and `@tpack-update-interval` must be set for update checking to activate.

## Self-update

When tpack is installed via auto-download or git clone, it can update itself from GitHub releases. It checks once every 24 hours.

To pin a specific version and disable self-update:

```bash
set -g @tpack-version '1.2.3'
```

!!! note
    Self-update only applies when tpack is installed via auto-download or git clone. If you installed tpack through a package manager (Homebrew, AUR, deb/rpm), use the package manager to update instead.
