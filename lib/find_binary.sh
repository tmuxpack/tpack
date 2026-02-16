# Shared helper for detecting the tpack binary.
# Usage: source this file, then call _find_tpack <tpm_root_dir>

FIND_BINARY_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$FIND_BINARY_DIR/download_binary.sh"

_find_tpack() {
	local root_dir="$1"

	if [ -x "$root_dir/dist/tpack" ]; then
		echo "$root_dir/dist/tpack"
		return
	fi
	if [ -x "$root_dir/tpack" ]; then
		echo "$root_dir/tpack"
		return
	fi
	if command -v tpack >/dev/null 2>&1; then
		command -v tpack
		return
	fi

	# Auto-download from GitHub Releases
	_download_tpack "$root_dir"
}
