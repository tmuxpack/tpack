# Configuration

This page walks through first-time setup: declaring plugins and loading tpack.

## 1. Open your tmux config

Your tmux configuration file is typically at one of these paths:

- `~/.tmux.conf`
- `$XDG_CONFIG_HOME/tmux/tmux.conf`

## 2. Declare plugins and initialize tpack

The configuration depends on how you installed tpack.

=== "Standalone binary"

    If you installed tpack via Homebrew, AUR, DEB/RPM, or built from source
    (i.e. the `tpack` binary is on your `$PATH`):

    ```bash
    # List of plugins
    set -g @plugin 'tmux-plugins/tmux-sensible'

    # Initialize tpack (keep this line at the very bottom of tmux.conf)
    run 'tpack init'
    ```

=== "Git clone (TPM drop-in)"

    If you cloned tpack into `~/.tmux/plugins/tpm`:

    ```bash
    # List of plugins
    set -g @plugin 'tmux-plugins/tmux-sensible'

    # Initialize tpack (keep this line at the very bottom of tmux.conf)
    run '~/.tmux/plugins/tpm/tpm'
    ```

    tpack automatically updates itself when using the auto-downloaded
    binary (the default for git clone installs) — no self-referencing
    plugin line is needed. If you built from source with `make build`,
    use `tpack self-update` or rebuild manually.

## 3. Reload tmux

```bash
tmux source ~/.tmux.conf
```

## 4. Install plugins

Press ++prefix+shift+i++ to fetch and install all declared plugins.

!!! tip
    After installation, tpack will display a message listing the plugins it installed.

## Plugin declaration syntax

| Format | Example | Description |
|---|---|---|
| `user/repo` | `tmux-plugins/tmux-sensible` | GitHub shorthand |
| `user/repo#branch` | `tmux-plugins/tmux-sensible#main` | Specific branch or tag |
| `https://github.com/user/repo.git` | `https://github.com/user/tmux-sensible.git` | Full HTTPS URL |
| `git@github.com:user/plugin` | `git@github.com:tmux-plugins/tmux-sensible` | Full git SSH URL (GitHub) |
| `git@bitbucket.com:user/plugin` | `git@bitbucket.com:user/tmux-plugin` | Non-GitHub git hosts |
| `user/plugin alias=name` | `tmux-plugins/tmux-sensible alias=sensible` | Custom directory name |

For a list of compatible plugins, see the [tmux-plugins list](https://github.com/tmux-plugins/list).

## Next step

Already a TPM user? See [Migrating from TPM](migrating-from-tpm.md). Otherwise, head to [Usage](../usage/index.md) to learn about key bindings, the TUI, and the CLI.
