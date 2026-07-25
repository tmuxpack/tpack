# Shared helper for detecting the tpack binary.
# Usage: source this file, then call _find_tpack <tpm_root_dir>

FIND_BINARY_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$FIND_BINARY_DIR/download_binary.sh"

_find_tpack() {
	local root_dir="$1"

	if [ -x "$root_dir/dist/tpack" ]; then
		printf '%s\n' "$root_dir/dist/tpack"
		return 0
	fi
	if [ -x "$root_dir/tpack" ]; then
		printf '%s\n' "$root_dir/tpack"
		return 0
	fi
	if command -v tpack >/dev/null 2>&1; then
		local tpack_path
		tpack_path="$(command -v tpack)"
		printf '%s\n' "$tpack_path"
		return 0
	fi

	# Auto-download from GitHub Releases
	if _download_tpack "$root_dir"; then
		printf '%s\n' "$root_dir/tpack"
		return 0
	fi
	return 1
}
