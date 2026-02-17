# Key Bindings

tpack ships with sensible default key bindings. All of them can be changed via tmux options.

## Default bindings

| Action | Key | tmux option |
|---|---|---|
| Install plugins | ++prefix+shift+i++ | `@tpack-install` |
| Update plugins | ++prefix+shift+u++ | `@tpack-update` |
| Clean/uninstall | ++prefix+alt+u++ | `@tpack-clean` |
| Open TUI | ++prefix+shift+t++ | `@tpack-tui` |

## Customization

Set the corresponding tmux option to a different key:

```bash
set -g @tpack-install 'I'   # default: I
set -g @tpack-update  'U'   # default: U
set -g @tpack-clean   'M-u' # default: M-u
set -g @tpack-tui     'T'   # default: T
```

!!! note
    The TPM-compatible options `@tpm-install`, `@tpm-update`, and `@tpm-clean` also work. tpack reads them as fallbacks when the `@tpack-*` equivalents are not set.
