# Installation

## Requirements

- **tmux** 1.9 or later
- **git**
- **bash**

## Install tpack

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

=== "Git clone"

    Clone the repository into the default TPM path:

    ```bash
    git clone https://github.com/tmuxpack/tpack ~/.tmux/plugins/tpm
    ```

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

## Verify

```bash
tpack --version
```

## Next step

Proceed to [Configuration](configuration.md) to add plugins to your tmux config.
