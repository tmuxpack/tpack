# Changing plugins install dir

By default, tpack installs plugins in a subfolder named `plugins/` inside
`$XDG_CONFIG_HOME/tmux/` if a `tmux.conf` file was found at that location, or
inside `~/.tmux/` otherwise.

You can change the install path by putting this in `.tmux.conf`:

    set-environment -g TMUX_PLUGIN_MANAGER_PATH '/some/other/path/'

Note: `TPACK_PLUGIN_PATH` is also supported as an alternative to `TMUX_PLUGIN_MANAGER_PATH`.

tpack initialization in `.tmux.conf` should also be updated:

    # initializes tpack in a new path
    run /some/other/path/tpm/tpm

Please make sure that the `run` line is at the very bottom of `.tmux.conf`.
