# Shared helper for detecting the tpm-go binary.
# Usage: source this file, then call _find_tpm_go <tpm_root_dir>

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
}
