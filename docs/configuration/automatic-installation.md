# Automatic Installation

Bootstrap tpack automatically when setting up a new machine from your dotfiles.

## Git clone setup

If you use the [git clone installation method](../getting-started/installation.md#option-b-git-clone-tpm-drop-in-replacement),
add this snippet to your `tmux.conf`, **before** the final `run` line:

```bash
if "test ! -d ~/.tmux/plugins/tpm" \
   "run 'git clone https://github.com/tmuxpack/tpack ~/.tmux/plugins/tpm && ~/.tmux/plugins/tpm/bin/install_plugins'"
```

This does two things:

1. Checks whether tpack is already installed at `~/.tmux/plugins/tpm`.
2. If not, clones the repository and runs the initial plugin installation.

On subsequent tmux starts the directory already exists, so the snippet is a no-op.

## Standalone binary setup

If you use the [standalone binary installation method](../getting-started/installation.md#option-a-standalone-binary),
bootstrap with your dotfiles manager or system provisioning tool (e.g. Ansible,
chezmoi, Homebrew Bundle). Then add this snippet to your `tmux.conf`, **before**
the final `run 'tpack init'` line:

```bash
if "command -v tpack" \
   "run 'tpack install'"
```

!!! tip
    This is especially useful when you sync your dotfiles across machines. New machines will bootstrap tpack and all your plugins on the first tmux launch.
