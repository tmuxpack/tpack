# Troubleshooting

Solutions to common issues with tpack.

??? question "Key bindings (prefix + I, prefix + U) are not defined"

    **Cause:** tmux version too old, or ZSH tmux plugin interfering.

    **Solution:**

    1. Check your tmux version: `tmux -V`. tpack requires tmux 1.9 or higher.
    2. If you have the ZSH tmux plugin installed, try disabling it to see if tpack works.

??? question "Plugins don't load when using tmux -f /path/to/config"

    **Cause:** The intended configuration is not among the active roots reported
    by the running tmux server, such as in an unusual or detached setup.

    **Solution:** tpack normally discovers every active `tmux -f` root
    automatically through tmux's `#{config_files}` format. If the intended file
    is not reported, make it the authoritative root with an absolute path:

    ```tmux
    set -g @tpack-config '/absolute/path/to/tmux.conf'
    ```

    Then reload that configuration with `tmux source-file /path/to/config`.

??? question "Strange characters appear when installing or updating plugins"

    **Cause:** The tmuxline.vim plugin can interfere with tpack's output.

    **Solution:** Uninstall tmuxline.vim and try again.

??? question "'failed to connect to server' when sourcing tmux.conf"

    **Cause:** Running `tmux source` outside of a tmux session.

    **Solution:** Make sure you run `tmux source ~/.tmux.conf` from inside a running tmux session.

??? question "tpack returned exit code 2 (Windows / Cygwin)"

    **Cause:** Windows line endings (CRLF) in plugin files, often caused by git's `core.autocrlf` setting.

    **Solution:** Convert all files to Unix line endings:

    ```bash
    find ~/.tmux -type d -name '.git*' -prune -o -type f -print0 | xargs -0 dos2unix
    ```

??? question "tpack returned exit code 127 (macOS with Homebrew tmux)"

    **Cause:** tmux's `run-shell` command uses a shell that doesn't read user configs, so Homebrew-installed binaries aren't found.

    **Solution:** Add the Homebrew prefix to tmux's PATH. Find your prefix:

    ```bash
    echo "$(brew --prefix)/bin"
    ```

    Then add this to `tmux.conf` **before** any `run` commands:

    ```bash
    set-environment -g PATH "/opt/homebrew/bin:/bin:/usr/bin"
    ```
