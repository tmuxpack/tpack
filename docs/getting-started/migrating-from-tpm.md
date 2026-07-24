# Migrating from TPM

tpack accepts TPM configuration syntax, so you can switch without rewriting
your plugin declarations. Its on-disk plugin layout is intentionally different.

!!! warning "One-way compatibility boundary"
    Switching back to TPM is not seamless after tpack migrates plugin
    directories. Do not run tpack and TPM against the same plugin root; using
    both managers there is unsupported. Scripts, shell configuration, or
    plugins with hard-coded paths such as `~/.tmux/plugins/tmux-sensible` may
    need to use the new canonical path or an explicit alias.

## What stays the same

Everything you already have continues to work:

- `set -g @plugin '...'` syntax
- `set -g @tpm_plugins '...'` legacy syntax
- `@tpm-install`, `@tpm-update`, `@tpm-clean` options
- `TMUX_PLUGIN_MANAGER_PATH` environment variable
- tpack launcher path `~/.tmux/plugins/tpm`
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

## One-time plugin directory migration

tpack identifies repositories by their normalized Git host and path, not only
their basename. Its generated directories resemble `tmux-87a1216f1f68`, are
ASCII, and are bounded to 64 bytes. This allows repositories such as
`first-owner/tmux` and `second-owner/tmux` to coexist in one plugin root.

Before an operational command changes plugins, tpack checks each old basename
directory's Git `origin`. It renames the directory to the canonical name only
when that origin exactly matches the configured repository identity. This is a
one-time, idempotent migration: later runs see the canonical directory and do
nothing. Explicit declarations such as `owner/repo alias=my-plugin` preserve
`my-plugin` as the directory and are not migrated.

Automatic migration does not guess, move, or delete a checkout when its origin
mismatches or cannot be read. A migration error stops the requested operation.
An unmatched legacy directory remains subject to normal orphan semantics, so a
later explicit `tpack clean` can remove it. Migration locking and destination
rechecks ensure concurrent tpack runs do not overwrite canonical directories.

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
    Existing exact-origin checkouts are reused, but their directories are
    renamed once as described above. Review hard-coded plugin paths before the
    first operational command.
