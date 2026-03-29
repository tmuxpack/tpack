# Installation

## Requirements

- **tmux** 1.9 or later
- **git**
- **bash**

There are two ways to install tpack. Pick whichever fits your workflow — both
give you the same features.

## Option A: Standalone binary

Install tpack as a binary on your `$PATH`. This is the recommended approach for
new setups. In your tmux config you'll use `run 'tpack init'` to load it.

=== "Homebrew"

    Works on macOS and Linux.

    ```bash
    brew install tmuxpack/tpack/tpack
    ```

=== "AUR"

    Arch Linux (via an AUR helper):

    ```bash
    yay -S tpack-bin
    ```

=== "DEB / RPM"

    Download the latest package from the [releases page](https://github.com/tmuxpack/tpack/releases/latest).

    Debian / Ubuntu:

    ```bash
    sudo dpkg -i tpack_*.deb
    ```

    Fedora / RHEL:

    ```bash
    sudo rpm -i tpack_*.rpm
    ```

=== "Go install"

    Requires [Go](https://go.dev/) 1.26 or later:

    ```bash
    go install github.com/tmuxpack/tpack/cmd/tpack@latest
    ```

    Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is on your `$PATH`.

=== "Build from source"

    ```bash
    git clone https://github.com/tmuxpack/tpack
    cd tpack
    make build
    ```

    The compiled binary is at `dist/tpack`. Move it somewhere on your `$PATH`:

    ```bash
    sudo cp dist/tpack /usr/local/bin/
    ```

## Option B: Git clone (TPM drop-in replacement)

Clone tpack into the TPM directory. This is fully backward compatible with
existing TPM configurations — if you're switching from TPM, no `tmux.conf`
changes are needed. In your tmux config you'll use
`run '~/.tmux/plugins/tpm/tpm'` to load it.

```bash
git clone https://github.com/tmuxpack/tpack ~/.tmux/plugins/tpm
```

## Verify

```bash
tpack version
```

## Next step

Proceed to [Configuration](configuration.md) to add plugins to your tmux config.
