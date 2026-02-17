# Interactive TUI

## Opening the TUI

Press ++prefix+shift+t++ to open the TUI. On tmux 3.2+ it launches in a popup window. On older versions it opens inline in the current pane.

## Plugin List

The default screen shows all declared plugins and their status (Installed, Not Installed, Outdated, Checking). Select plugins with ++space++ or ++tab++, then trigger an operation.

## Progress View

Displayed during install, update, or clean operations. Shows real-time per-plugin progress with a progress bar and status indicators. After completion, use ++arrow-up++ / ++arrow-down++ to browse results and ++enter++ to view commits for updated plugins.

## Commit History

After an update, select a result and press ++enter++ to see the recent commits pulled in for that plugin. Press ++escape++ to return to the progress view.

## Debug View

Press ++at++ on the plugin list to open the debug screen. Displays tpack version, binary path, and configuration details useful for troubleshooting.

## Navigation reference

| Key | Action |
|-----|--------|
| ++arrow-up++ / ++k++ | Move cursor up |
| ++arrow-down++ / ++j++ | Move cursor down |
| ++space++ / ++tab++ | Toggle selection |
| ++i++ | Install selected / not-installed plugins |
| ++u++ | Update selected / installed plugins |
| ++c++ | Clean orphaned plugin directories |
| ++x++ | Uninstall selected plugins |
| ++enter++ | View commits for the selected result |
| ++escape++ | Go back to previous screen |
| ++q++ | Quit |
| ++ctrl+c++ | Force quit |
