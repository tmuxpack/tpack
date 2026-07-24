# Migration Restart Notification Design

## Goal

Notify users when tpack's one-time plugin migration renames one or more plugin
folders, because a running tmux server or plugin may still reference an old
path. The notification should be brief, generic, and visible from `tpack init`.

## Behavior

After at least one successful legacy-folder rename, tpack emits exactly one
warning through the existing output path:

> Plugin folders were renamed during migration; restart tmux if plugins fail to load.

No warning is emitted when migration performs no renames. Multiple renames in
one migration produce one warning. If an earlier plugin is renamed and a later
plugin migration fails, tpack emits the warning before reporting the migration
error because the earlier path change has already occurred.

Warning severity ensures the message reaches the tmux status line during
`tpack init`, as well as stderr or tmux command output in other operational
commands. The warning is naturally one-time because subsequent idempotent
migration runs find the canonical folders and perform no renames.

## Design

`plug.MigrateLegacy` will return both whether any rename succeeded and an error.
It will retain the successful-rename result if a later plugin fails. This keeps
the filesystem migration independent from user-interface concerns.

`config.LoadPlugins` will inspect that result and invoke its existing warning
callback once when a rename occurred. It will do so before returning any
migration error. Existing command output routing then controls where the
warning is displayed without command-specific migration logic.

No persistent notification marker, new configuration, or folder-name listing
will be added.

## Error Handling

Existing migration failure behavior remains unchanged: the requested operation
stops and receives the wrapped migration error. The warning itself remains
non-fatal. A nil warning callback is supported and does not affect migration or
error reporting.

## Testing

Tests will verify that:

- a successful rename reports that migration changed a path;
- a no-op migration does not report a change;
- multiple successful renames produce one warning;
- an earlier rename followed by a migration failure still produces the warning;
- `LoadPlugins` emits the exact warning only when a rename occurred; and
- existing migration errors and idempotence remain intact.
