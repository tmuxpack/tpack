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
| `tpack completion [bash\|zsh\|fish]` | Generate shell completion scripts |

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

## Shell Completions

Generate and install shell completion scripts for tab-completion support.

### Zsh

```bash
tpack completion zsh > "${fpath[1]}/_tpack"
compinit
```

### Bash

```bash
# Linux:
tpack completion bash > /etc/bash_completion.d/tpack

# macOS:
tpack completion bash > $(brew --prefix)/etc/bash_completion.d/tpack
```

### Fish

```bash
tpack completion fish > ~/.config/fish/completions/tpack.fish
```

Completions include all commands, flags, and dynamic plugin name completion for `tpack update` and `tpack commits --name`.

## Legacy shell scripts

For backward compatibility with TPM, tpack ships shell script wrappers in `bin/`. These work if tpack is installed via git clone to `~/.tmux/plugins/tpm`:

```bash
~/.tmux/plugins/tpm/bin/install_plugins
~/.tmux/plugins/tpm/bin/update_plugins all
~/.tmux/plugins/tpm/bin/update_plugins tmux-sensible
~/.tmux/plugins/tpm/bin/clean_plugins
```
