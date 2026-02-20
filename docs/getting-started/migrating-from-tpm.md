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
| Plugin Registry | Browse a curated plugin registry from the TUI. Search, filter by category, and install with a single key press. |
| CLI | The `tpack` binary provides command-line management outside of tmux. |
| Automatic update checking | tpack can check for plugin updates in the background and prompt or auto-update. |
| Color-customizable TUI | Configure colors for the TUI via tmux options. |
| Self-update | tpack can update itself without manual intervention. |

## How to switch

=== "Change the git remote"

    If you installed TPM with `git clone`, you can switch in place — no config
    changes needed:

    ```bash
    cd ~/.tmux/plugins/tpm
    git remote set-url origin https://github.com/tmuxpack/tpack
    git pull
    ```

    The `run '~/.tmux/plugins/tpm/tpm'` line in your `tmux.conf` stays the same.

    Reload tmux to pick up the new binary:

    ```bash
    tmux source ~/.tmux.conf
    ```

=== "Install via package manager"

    If you prefer installing through a package manager (Homebrew, AUR, DEB/RPM)
    or `go install`:

    1. Install tpack using any method from the [Installation](installation.md)
       page.
    2. Replace the `run` line at the bottom of your `tmux.conf`:

        ```bash
        # Remove or comment out the old line:
        # run '~/.tmux/plugins/tpm/tpm'

        # Add:
        run 'tpack init'
        ```

    3. Optionally remove the old TPM directory:

        ```bash
        rm -rf ~/.tmux/plugins/tpm
        ```

    4. Reload tmux:

        ```bash
        tmux source ~/.tmux.conf
        ```

!!! note
    Your existing plugins stay in place. tpack will manage them going forward exactly as TPM did, plus you get the new features listed above.
