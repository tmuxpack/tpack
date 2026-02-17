# CLI Reference

The `tpack` binary can be used directly from the command line, outside of tmux key bindings.

## Commands

| Command | Description |
|---------|-------------|
| `tpack install` | Install all plugins declared in tmux.conf |
| `tpack update [name]` | Update a specific plugin, or all if `all` is given |
| `tpack clean` | Remove plugin directories not declared in tmux.conf |
| `tpack source` | Source all plugins without installing (useful for already-cloned plugins) |
| `tpack tui` | Open the interactive TUI |
| `tpack commits` | Show commit history for a plugin (internal, used by the TUI) |
| `tpack check-updates` | Check if any plugins have updates available |
| `tpack self-update` | Update the tpack binary to the latest release |
| `tpack version` | Print tpack version |
| `tpack init` | Initialize tpack (backward compatibility with TPM scripts) |

## Examples

Install all declared plugins:

```bash
tpack install
```

Update a single plugin:

```bash
tpack update tmux-sensible
```

Update all plugins:

```bash
tpack update all
```

Remove orphaned plugin directories:

```bash
tpack clean
```

## Legacy shell scripts

For backward compatibility with TPM, tpack ships shell script wrappers in `bin/`. These work if tpack is installed via git clone to `~/.tmux/plugins/tpm`:

```bash
~/.tmux/plugins/tpm/bin/install_plugins
~/.tmux/plugins/tpm/bin/update_plugins all
~/.tmux/plugins/tpm/bin/update_plugins tmux-sensible
~/.tmux/plugins/tpm/bin/clean_plugins
```
