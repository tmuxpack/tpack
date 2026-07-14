# FAQ

## Can I use tpack alongside TPM?

No, tpack replaces TPM. They both install to `~/.tmux/plugins/tpm`. But your existing `tmux.conf` configuration doesn't need to change — tpack is backward compatible.

## Which tmux version do I need?

Minimum: tmux 1.9. For TUI popup support, tmux 3.2 or later is recommended. On older versions, the TUI opens inline instead of in a popup.

## Where are plugins installed?

Existing installs use `~/.tmux/plugins/` or `$XDG_CONFIG_HOME/tmux/plugins/`; fresh installs default to `~/.local/share/tmux/plugins/`. This can be changed with the `@tpack-plugin-path` option — see [Plugin Directory](../configuration/plugin-directory.md).

## How do I check which plugins are installed?

Open the TUI with ++prefix+shift+t++ to see all installed plugins. Or list your plugins directory (see [Plugin Directory](../configuration/plugin-directory.md) for where it is on your setup).

## Does tpack work with existing TPM plugins?

Yes. All plugins from the [tmux-plugins](https://github.com/tmux-plugins/list) ecosystem are fully compatible. tpack uses the same plugin format as TPM.

## How do I specify a plugin branch or tag?

Use the `#branch` suffix: `set -g @plugin 'user/repo#branch'`

## How do I discover new plugins?

Open the TUI with ++prefix+shift+t++ and press ++b++ to browse a curated plugin registry. You can search by name with ++slash++ and filter by category with ++tab++. Press ++i++ on any plugin to install it — the entry is added to your `tmux.conf` automatically.

## Why does clean remove everything when I removed all @plugin lines?

If your `tmux.conf` is readable and declares zero `@plugin` lines, `clean` removes every installed plugin directory (except tpm/tpack itself) — matching TPM's original behavior. A readable, empty declared-plugin list is treated as "I want nothing installed," not as a missing config.

A missing or unreadable `tmux.conf` is a different, explicit case: tpack refuses to run and reports a config error instead of guessing. This distinction exists so a permissions problem or a deleted `tmux.conf` can never be silently treated as "no plugins" and wipe your plugin directory by accident. Note that this guarantee covers `tmux.conf` itself: files pulled in via `source` are read best-effort (conditional sourcing is legal), so keep your `@plugin` lines in `tmux.conf` if you want the strongest protection.

## How do I update tpack itself?

If installed via git clone or auto-download, tpack self-updates automatically (checks once every 24 hours). To update manually: `tpack self-update`. If installed via a package manager, use that package manager instead (e.g., `brew upgrade tpack`).
