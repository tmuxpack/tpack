# Managing Plugins

tpack binds three key combinations in tmux for the core plugin lifecycle. Each opens the TUI with the corresponding operation pre-selected.

## Installing plugins

1. Add a plugin line to your tmux config:

    ```bash
    set -g @plugin 'tmux-plugins/tmux-sensible'
    ```

2. Press ++prefix+shift+i++ (capital I, as in **I**nstall).

3. The plugin is cloned to `~/.tmux/plugins/` and sourced automatically.

## Updating plugins

Press ++prefix+shift+u++ to update plugins. The TUI opens and you can select which plugins to update.

## Uninstalling plugins

1. Remove (or comment out) the plugin line from your tmux config.

2. Press ++prefix+alt+u++ (lowercase u, as in **u**ninstall).

3. The orphaned plugin directory is removed.

!!! tip
    All three key bindings can be customized. See [Key Bindings](../configuration/key-bindings.md) for details.
