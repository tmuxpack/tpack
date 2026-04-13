# Configuration

tpack is configured entirely through tmux options in your `tmux.conf`. No
separate config files needed.

---

**[Key Bindings](key-bindings.md)** — Customize the keyboard shortcuts for
installing, updating, and cleaning plugins.

**[Colors](colors.md)** — Adjust the TUI color palette to match your terminal
theme.

**[Plugin Directory](plugin-directory.md)** — Change where tpack installs
plugins on disk.

**[Automatic Installation](automatic-installation.md)** — Bootstrap tpack on new
machines from your dotfiles.

## Hiding browse categories

The browse screen displays every category advertised by the plugin registry.
Hide ones you're not interested in with `@tpack-hidden-categories`, a
comma-separated list of category names:

```bash
set -g @tpack-hidden-categories 'ai'
```

Hidden categories disappear from the category bar, and their plugins are
excluded from the `all` and `new` tabs. Names must match the registry spelling
exactly.

## Pinning the tpack version

Pin tpack to a specific release (disables self-update):

```bash
set -g @tpack-version '1.2.3'
```

See [Automatic Updates](../usage/automatic-updates.md#self-update) for details.
