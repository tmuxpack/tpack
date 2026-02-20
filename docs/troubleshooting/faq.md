# FAQ

## Can I use tpack alongside TPM?

No, tpack replaces TPM. They both install to `~/.tmux/plugins/tpm`. But your existing `tmux.conf` configuration doesn't need to change — tpack is backward compatible.

## Which tmux version do I need?

Minimum: tmux 1.9. For TUI popup support, tmux 3.2 or later is recommended. On older versions, the TUI opens inline instead of in a popup.

## Where are plugins installed?

By default, plugins are installed to `~/.tmux/plugins/`. This can be changed — see [Plugin Directory](../configuration/plugin-directory.md).

## How do I check which plugins are installed?

Open the TUI with ++prefix+shift+t++ to see all installed plugins. Or list the plugins directory: `ls ~/.tmux/plugins/`.

## Does tpack work with existing TPM plugins?

Yes. All plugins from the [tmux-plugins](https://github.com/tmux-plugins/list) ecosystem are fully compatible. tpack uses the same plugin format as TPM.

## How do I specify a plugin branch or tag?

Use the `#branch` suffix: `set -g @plugin 'user/repo#branch'`

## How do I discover new plugins?

Open the TUI with ++prefix+shift+t++ and press ++b++ to browse a curated plugin registry. You can search by name with ++slash++ and filter by category with ++tab++. Press ++i++ on any plugin to install it — the entry is added to your `tmux.conf` automatically.

## How do I update tpack itself?

If installed via git clone or auto-download, tpack self-updates automatically (checks once every 24 hours). To update manually: `tpack self-update`. If installed via a package manager, use that package manager instead (e.g., `brew upgrade tpack`).
