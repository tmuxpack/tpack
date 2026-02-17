# Configuration

This page walks through first-time setup: declaring plugins and loading tpack.

## 1. Open your tmux config

Your tmux configuration file is typically at one of these paths:

- `~/.tmux.conf`
- `$XDG_CONFIG_HOME/tmux/tmux.conf`

## 2. Declare plugins

Add `set -g @plugin` lines for each plugin you want:

```bash
# List of plugins
set -g @plugin 'tmuxpack/tpack'
set -g @plugin 'tmux-plugins/tmux-sensible'
```

## 3. Add the initialization line

This **must** be the very last line in your config:

```bash
# Initialize tpack (keep this line at the very bottom of tmux.conf)
run '~/.tmux/plugins/tpm/tpm'
```

## 4. Reload tmux

```bash
tmux source ~/.tmux.conf
```

## 5. Install plugins

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
