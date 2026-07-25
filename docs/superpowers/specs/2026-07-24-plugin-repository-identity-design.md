# Plugin Repository Identity and Migration Design

## Context

Issue #31 reports that repositories with the same basename are treated as one
plugin. For example, `catppuccin/tmux`, `dracula/tmux`, and
`nordtheme/tmux` all become `tmux`. The basename currently serves as the
display name, installed-plugin identity, and installation directory. This
causes false installed labels in the browser and prevents replacing or
co-installing same-basename repositories.

The existing `alias=` syntax can avoid a collision manually, but browser
installs do not generate aliases and ordinary plugin declarations remain
ambiguous.

## Goals

- Distinguish repositories by normalized Git repository identity.
- Support multiple same-basename repositories simultaneously.
- Detect a changed repository declaration and install the new repository.
- Give every unaliased repository a deterministic, bounded installation
  directory.
- Automatically and safely migrate existing basename directories once.
- Keep explicit aliases authoritative.
- Document the intentional break from TPM's directory convention.

## Non-Goals

- Preserve the ability to switch back to TPM without filesystem changes.
- Run TPM and tpack against the same plugin directory.
- Delete or overwrite legacy directories during migration.
- Introduce persistent repository-to-directory metadata.

## Base Branch and Tooling

Implementation must branch from `refactor/folder_conventions` at or after
commit `c51073c`, not from `main`.

OpenCode's Go LSP prerequisite is already satisfied: global configuration has
`"lsp": true`, `gopls` v0.22.0 is available at `/usr/bin/gopls`, and an
OpenCode diagnostic request successfully loads `gopls` for this repository.

## Repository Identity

`plug.Plugin` will separate concepts currently conflated in `Name`:

- `Spec`: the clone source without a ref, as today.
- `Identity`: normalized `host/path`, used for repository equality.
- `Name`: a human-facing repository path such as `catppuccin/tmux`.
- `DirName`: the safe directory component used for filesystem operations.
- `Alias`: the optional explicit directory override, as today.

Normalization will make GitHub shorthand, HTTPS, SSH, and SCP-style forms of
the same repository equal. It will remove credentials and a trailing `.git`,
normalize the host, and retain enough host/path information to distinguish
repositories on different hosts or under different owners. Refs remain
separate from repository identity.

Invalid or unidentifiable repository specifications fail configuration loading
with a message naming the offending declaration. Normalization is pure and
does not access the network.

## Installation Directory Naming

Each unaliased plugin receives a deterministic generated directory name:

```text
<sanitized-and-truncated-repository-basename>-<12-hex-SHA256-prefix>
```

The SHA-256 input is the full normalized identity. The complete generated name
is capped at 64 ASCII bytes; the readable prefix is truncated as needed. An
empty sanitized basename falls back to `plugin`. For example, the theme
repositories will each use a distinct short directory resembling
`tmux-4f83c2a1e9c7`.

An explicit `alias=` value remains the exact `DirName` and opts that plugin out
of generated-name migration. All directory names continue through
`plug.Root.Child` validation.

Configuration loading rejects different identities that resolve to the same
`DirName`. This covers conflicting aliases and detects the highly unlikely
generated hash-prefix collision instead of silently conflating plugins.

## Runtime Behavior

All filesystem operations use `DirName`, including install detection, clone,
update, source, uninstall, remove, clean, load-error association, and update
checks. User-facing lists, messages, and completion use `Name` where a
repository label is expected.

The TUI plugin item carries `Identity` and `DirName`. The browser marks a
registry entry installed only when its normalized identity equals a configured
plugin identity. The basename fallback is removed.

Installing from the browser parses the selected registry entry through the
same identity and naming path used by configuration parsing. It appends the
original clone spec to `tmux.conf`, displays the repository name, and queues
the operation against the generated `DirName`.

Orphan discovery compares top-level installed directory names with declared
`DirName` values. A mismatched legacy checkout is left alone by migration and
is removed only if the user later invokes the existing clean operation.

## Automatic Migration

Plugin parsing remains side-effect free. A shared operational loader gathers
plugins and runs migration before install, update, source, clean, update
checks, initialization, or TUI startup. Shell completion parses plugin names
without migrating the filesystem.

For each plugin, migration performs these steps:

1. Skip plugins with an explicit alias.
2. Skip when the canonical `DirName` directory already exists.
3. Look for the legacy basename directory.
4. Read that checkout's Git `origin` through a small `OriginReader` interface.
5. Normalize the origin and compare it with the configured identity.
6. On an exact match, atomically rename the legacy directory to `DirName`.
7. On an identity mismatch, leave the legacy directory untouched and consider
   the configured repository uninstalled.
8. On origin inspection or rename failure, stop with an actionable migration
   error instead of guessing, overwriting, or deleting data.

When several declarations share a basename, at most one can match and migrate
the legacy checkout. The others remain uninstalled and receive independent
canonical directories when installed. Migration is naturally idempotent: a
successful rename makes later runs skip it, so no state marker is required.

If both canonical and legacy directories exist, migration does not alter
either. Normal orphan behavior can clean the legacy directory later.

## Error Handling and Safety

- Migration never deletes or overwrites a directory.
- A root-scoped migration lock plus destination rechecks prevent concurrent
  tpack processes from overwriting a canonical directory. Arbitrary external
  writers that create a destination during the final atomic-rename window are
  outside this guarantee.
- A failed migration prevents subsequent operations in that command.
- Errors identify the source, destination, and underlying Git or filesystem
  failure where applicable.
- Identity mismatches are not migration errors; they represent a different
  repository and leave the legacy checkout untouched.
- Generated and aliased directory collisions are configuration errors.
- Existing plugin-root validation and symlink safety remain in force.

## TDD Strategy

Implementation follows strict red-green-refactor slices, with each production
change preceded by a witnessed failing test:

1. Add table tests for identity normalization across shorthand, HTTPS, SSH,
   SCP, credentials, `.git`, owners, hosts, and refs.
2. Add tests for deterministic bounded `DirName` values, safe sanitization,
   aliases, and directory collisions.
3. Implement only enough identity and naming behavior to pass those tests.
4. Add migration tests for an exact-origin rename, origin mismatch, existing
   canonical destination, explicit alias, multiple same-basename plugins,
   origin-read failure, rename failure, and repeat-run idempotence.
5. Implement `OriginReader`, its Git CLI adapter, and the migrator.
6. Add manager and TUI regression tests reproducing issue #31: distinct theme
   paths, installability after replacement, and exact browser installed state.
7. Route operational plugin loading through migration and update source,
   update, clean, orphan, load-error, and completion tests for the split name
   and directory concepts.
8. Add an integration test that migrates a legacy checkout and then installs
   two repositories sharing a basename.
9. Update documentation and run the complete verification suite.

Required final verification:

```bash
make lint
make test
```

## Documentation

`README.md` and the migration guide will state that tpack now uses canonical
repository-specific directories and automatically renames recognized legacy
installs. The note will explain that this resolves same-basename collisions but
breaks seamless switching back to TPM and may require updates to user scripts
that hardcode plugin directory paths. Explicit aliases retain their configured
directory names.
