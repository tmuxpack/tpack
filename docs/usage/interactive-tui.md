# Interactive TUI

## Opening the TUI

Press ++prefix+shift+t++ to open the TUI. On tmux 3.2+ it launches in a popup window. On older versions it opens inline in the current pane.

## Plugin List

The default screen shows all declared plugins and their status (Installed, Not Installed, Outdated, Checking). Select plugins with ++space++ or ++tab++, then trigger an operation.

## Progress View

Displayed during install, update, or clean operations. Shows real-time per-plugin progress with a progress bar and status indicators. After completion, use ++arrow-up++ / ++arrow-down++ to browse results and ++enter++ to view commits for updated plugins.

## Commit History

After an update, select a result and press ++enter++ to see the recent commits pulled in for that plugin. Press ++escape++ to return to the progress view.

## Browse Screen

Press ++b++ on the plugin list to open the browse screen. It fetches a curated plugin registry and displays available plugins with star counts and descriptions. Plugins already in your configuration are marked as "(installed)".

### Category filtering

A category bar at the top shows all available categories. Press ++tab++ to cycle through them. The default view shows all plugins.

### Searching

Press ++slash++ to enter search mode. Type to filter plugins by name or description in real time. Press ++enter++ to accept the filter or ++escape++ to revert to the previous query.

### Installing from browse

Select a plugin and press ++i++ to install it. The plugin line is automatically added to your `tmux.conf` and installation begins immediately — no manual config editing required.

### Opening a plugin URL

Press ++enter++ on a plugin to open its GitHub page in your browser. The URL is also copied to the clipboard.

## Debug View

Press ++at++ on the plugin list to open the debug screen. Displays tpack version, binary path, and configuration details useful for troubleshooting.

## Navigation reference

### Plugin list

| Key | Action |
|-----|--------|
| ++arrow-up++ / ++k++ | Move cursor up |
| ++arrow-down++ / ++j++ | Move cursor down |
| ++space++ / ++tab++ | Toggle selection |
| ++i++ | Install selected / not-installed plugins |
| ++u++ | Update selected / installed plugins |
| ++c++ | Clean orphaned plugin directories |
| ++x++ | Uninstall selected plugins |
| ++b++ | Open browse screen |
| ++at++ | Open debug view |
| ++q++ | Quit |
| ++ctrl+c++ | Force quit |

### Browse screen

| Key | Action |
|-----|--------|
| ++arrow-up++ / ++k++ | Move cursor up |
| ++arrow-down++ / ++j++ | Move cursor down |
| ++slash++ | Search / filter plugins |
| ++tab++ | Cycle category |
| ++i++ | Install selected plugin |
| ++enter++ | Open plugin URL in browser |
| ++escape++ | Go back to plugin list |
| ++q++ | Quit |

### Progress view

| Key | Action |
|-----|--------|
| ++arrow-up++ / ++arrow-down++ | Browse results |
| ++enter++ | View commits for the selected result |
| ++escape++ | Go back to plugin list |
