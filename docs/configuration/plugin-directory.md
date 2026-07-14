# Plugin Directory

tpack resolves the plugin directory in this order — the first match wins:

1. The `@tpack-plugin-path` tmux option
2. The `TPACK_PLUGIN_PATH` tmux environment variable
3. The `TMUX_PLUGIN_MANAGER_PATH` tmux environment variable (TPM compatibility)
4. An existing plugin directory: `$XDG_CONFIG_HOME/tmux/plugins/` or `~/.tmux/plugins/`,
   preferring the one next to your `tmux.conf`
5. `$XDG_DATA_HOME/tmux/plugins/` (usually `~/.local/share/tmux/plugins/`)

Existing installs keep working unchanged: if you already have plugins in
`~/.tmux/plugins/` or `$XDG_CONFIG_HOME/tmux/plugins/`, step 4 finds them and
nothing moves. The XDG data directory default (step 5) only applies to fresh
installs with no plugin directory on disk, following the
[XDG Base Directory Specification](https://specifications.freedesktop.org/basedir/latest/) —
installed plugins are data, so they belong under `$XDG_DATA_HOME` rather than
in your config directory (and out of your dotfiles repo).

## Overriding the path

Set the `@tpack-plugin-path` option in your `tmux.conf`:

```bash
set -g @tpack-plugin-path '~/some/other/path/'
```

`~`, `$HOME`, and `$XDG_CONFIG_HOME` are expanded.

For TPM compatibility, the `TMUX_PLUGIN_MANAGER_PATH` and `TPACK_PLUGIN_PATH`
environment variables still work:

```bash
set-environment -g TMUX_PLUGIN_MANAGER_PATH '/some/other/path/'
```

`@tpack-plugin-path` takes priority over both environment variables.

When changing the path with a git-clone install, update the initialization line
at the bottom of your config to match:

```bash
run /some/other/path/tpm/tpm
```

!!! warning
    Keep the `run` line at the very bottom of `tmux.conf`. tpack must load after all plugin declarations.
