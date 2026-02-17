# Plugin Directory

By default, tpack installs plugins to a `plugins/` subfolder inside your tmux config directory:

- `$XDG_CONFIG_HOME/tmux/plugins/` — if a `tmux.conf` was found there
- `~/.tmux/plugins/` — otherwise

## Overriding the path

Set the `TMUX_PLUGIN_MANAGER_PATH` environment variable in your `tmux.conf`:

```bash
set-environment -g TMUX_PLUGIN_MANAGER_PATH '/some/other/path/'
```

`TPACK_PLUGIN_PATH` also works and takes priority over `TMUX_PLUGIN_MANAGER_PATH` when both are set.

When changing the path, update the initialization line at the bottom of your config to match:

```bash
run /some/other/path/tpm/tpm
```

!!! warning
    Keep the `run` line at the very bottom of `tmux.conf`. tpack must load after all plugin declarations.
