# tpack - Tmux Plugin Manager

Installs and loads `tmux` plugins.

Tested and working on Linux, OSX, and Cygwin.

See list of plugins [here](https://github.com/tmux-plugins/list).

### Installation

Requirements: `tmux` version 1.9 (or higher), `git`, `bash`.

Clone tpack:

```bash
git clone https://github.com/tmuxpack/tpack ~/.tmux/plugins/tpm
```

Put this at the bottom of `~/.tmux.conf` (`$XDG_CONFIG_HOME/tmux/tmux.conf`
works too):

```bash
# List of plugins
set -g @plugin 'tmuxpack/tpack'
set -g @plugin 'tmux-plugins/tmux-sensible'

# Other examples:
# set -g @plugin 'github_username/plugin_name'
# set -g @plugin 'github_username/plugin_name#branch'
# set -g @plugin 'git@github.com:user/plugin'
# set -g @plugin 'git@bitbucket.com:user/plugin'

# Initialize tpack (keep this line at the very bottom of tmux.conf)
run '~/.tmux/plugins/tpm/tpm'
```

Reload TMUX environment so tpack is sourced:

```bash
# type this in terminal if tmux is already running
tmux source ~/.tmux.conf
```

That's it!

### Installing plugins

1. Add new plugin to `~/.tmux.conf` with `set -g @plugin '...'`
2. Press `prefix` + <kbd>I</kbd> (capital i, as in **I**nstall) to fetch the plugin.

You're good to go! The plugin was cloned to `~/.tmux/plugins/` dir and sourced.

### Uninstalling plugins

1. Remove (or comment out) plugin from the list.
2. Press `prefix` + <kbd>alt</kbd> + <kbd>u</kbd> (lowercase u as in **u**ninstall) to remove the plugin.

All the plugins are installed to `~/.tmux/plugins/` so alternatively you can
find plugin directory there and remove it.

### Key bindings

`prefix` + <kbd>I</kbd>
- Installs new plugins from GitHub or any other git repository
- Refreshes TMUX environment

`prefix` + <kbd>U</kbd>
- updates plugin(s)

`prefix` + <kbd>alt</kbd> + <kbd>u</kbd>
- remove/uninstall plugins not on the plugin list

### Docs

- [Help, tpack not working](docs/tpack_not_working.md) - problem solutions

More advanced features and instructions, regular users probably do not need
this:

- [How to create a plugin](docs/how_to_create_plugin.md). It's easy.
- [Managing plugins via the command line](docs/managing_plugins_via_cmd_line.md)
- [Changing plugins install dir](docs/changing_plugins_install_dir.md)
- [Automatic tpack installation on a new machine](docs/automatic_tpack_installation.md)

### Tests

Run tests with:

```bash
make test
```

### License

[MIT](LICENSE.md)
