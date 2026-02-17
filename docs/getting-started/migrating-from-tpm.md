# Migrating from TPM

tpack is a drop-in replacement for TPM. You can switch without changing your `tmux.conf`.

## What stays the same

Everything you already have continues to work:

- `set -g @plugin '...'` syntax
- `set -g @tpm_plugins '...'` legacy syntax
- `@tpm-install`, `@tpm-update`, `@tpm-clean` options
- `TMUX_PLUGIN_MANAGER_PATH` environment variable
- Install path `~/.tmux/plugins/tpm`
- Key bindings:
    - ++prefix+shift+i++ — install plugins
    - ++prefix+shift+u++ — update plugins
    - ++prefix+alt+u++ — clean (remove unused) plugins

## What's new

tpack adds features that TPM does not have:

| Feature | Details |
|---|---|
| Interactive TUI | ++prefix+shift+t++ opens a terminal UI for browsing, installing, updating, and removing plugins. |
| CLI | The `tpack` binary provides command-line management outside of tmux. |
| Automatic update checking | tpack can check for plugin updates in the background and prompt or auto-update. |
| Color-customizable TUI | Configure colors for the TUI via tmux options. |
| Self-update | tpack can update itself without manual intervention. |

## How to switch

1. Install tpack using any method from the [Installation](installation.md) page. tpack installs to `~/.tmux/plugins/tpm` — the same path TPM uses — so it replaces TPM automatically.
2. Reload your tmux config:

    ```bash
    tmux source ~/.tmux.conf
    ```

That's it. No config changes needed.

!!! note
    Your existing plugins stay in place. tpack will manage them going forward exactly as TPM did, plus you get the new features listed above.
