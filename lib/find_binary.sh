# Shared helper for detecting the tpm-go binary.
# Usage: source this file, then call _find_tpm_go <tpm_root_dir>

FIND_BINARY_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
source "$FIND_BINARY_DIR/download_binary.sh"

_find_tpm_go() {
	local root_dir="$1"

	if [ -x "$root_dir/dist/tpm-go" ]; then
		echo "$root_dir/dist/tpm-go"
		return
	fi
	if [ -x "$root_dir/tpm-go" ]; then
		echo "$root_dir/tpm-go"
		return
	fi
	if command -v tpm-go >/dev/null 2>&1; then
		command -v tpm-go
		return
	fi

	# Auto-download from GitHub Releases
	_download_tpm_go "$root_dir"
}
