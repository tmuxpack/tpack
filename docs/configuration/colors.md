# Colors

Customize the TUI color palette with tmux options.

## Color options

| Option | Default | Description |
|---|---|---|
| `@tpack-color-primary` | `#89b4fa` | Headers, active selections, focused elements |
| `@tpack-color-secondary` | `#a6e3a1` | Success states, confirmations |
| `@tpack-color-accent` | `#f9e2af` | Highlights, warnings, attention items |
| `@tpack-color-error` | `#f38ba8` | Error messages, failed operations |
| `@tpack-color-muted` | `#6c7086` | Disabled items, secondary text, borders |
| `@tpack-color-text` | `#cdd6f4` | Main body text |

## Full example

```bash
# Custom color palette
set -g @tpack-color-primary   '#89b4fa'
set -g @tpack-color-secondary '#a6e3a1'
set -g @tpack-color-accent    '#f9e2af'
set -g @tpack-color-error     '#f38ba8'
set -g @tpack-color-muted     '#6c7086'
set -g @tpack-color-text      '#cdd6f4'
```

!!! note
    Colors use hex format. The defaults follow the [Catppuccin Mocha](https://github.com/catppuccin/catppuccin) palette.
