# Auto-download tpm-go binary from GitHub Releases.
# Usage: source this file, then call _download_tpm_go <tpm_root_dir>
# Returns the path to the downloaded binary on success, empty on failure.

_TPM_GITHUB_REPO="AntoineGS/tpm"

_download_tpm_go() {
	local root_dir="$1"

	# Opt-out via environment variable
	if [ "${TPM_AUTO_DOWNLOAD:-}" = "0" ]; then
		return
	fi

	# Detect OS
	local os
	case "$(uname -s)" in
		Linux)  os="linux" ;;
		Darwin) os="darwin" ;;
		FreeBSD) os="freebsd" ;;
		*) return ;;
	esac

	# Detect architecture
	local arch
	case "$(uname -m)" in
		x86_64)  arch="amd64" ;;
		aarch64|arm64) arch="arm64" ;;
		*) return ;;
	esac

	# Determine download tool
	local download_cmd=""
	if command -v curl >/dev/null 2>&1; then
		download_cmd="curl"
	elif command -v wget >/dev/null 2>&1; then
		download_cmd="wget"
	else
		return
	fi

	# Find latest release version via GitHub redirect
	local version=""
	if [ "$download_cmd" = "curl" ]; then
		version=$(curl -sI "https://github.com/${_TPM_GITHUB_REPO}/releases/latest" 2>/dev/null \
			| grep -i '^location:' | sed 's|.*/tag/v\{0,1\}||' | tr -d '[:space:]')
	else
		version=$(wget --server-response --max-redirect=0 "https://github.com/${_TPM_GITHUB_REPO}/releases/latest" 2>&1 \
			| grep -i 'location:' | sed 's|.*/tag/v\{0,1\}||' | tr -d '[:space:]')
	fi

	if [ -z "$version" ]; then
		return
	fi

	# Download archive to temp file
	local archive_name="tpm-go_${version}_${os}_${arch}.tar.gz"
	local url="https://github.com/${_TPM_GITHUB_REPO}/releases/download/v${version}/${archive_name}"
	local tmp_dir
	tmp_dir=$(mktemp -d 2>/dev/null) || return
	local tmp_archive="${tmp_dir}/${archive_name}"

	if [ "$download_cmd" = "curl" ]; then
		curl -sL -o "$tmp_archive" "$url" 2>/dev/null || { rm -rf "$tmp_dir"; return; }
	else
		wget -q -O "$tmp_archive" "$url" 2>/dev/null || { rm -rf "$tmp_dir"; return; }
	fi

	# Verify we got a real file (not an HTML error page)
	if [ ! -s "$tmp_archive" ]; then
		rm -rf "$tmp_dir"
		return
	fi

	# Extract tpm-go binary
	tar -xzf "$tmp_archive" -C "$tmp_dir" tpm-go 2>/dev/null || { rm -rf "$tmp_dir"; return; }

	if [ ! -f "$tmp_dir/tpm-go" ]; then
		rm -rf "$tmp_dir"
		return
	fi

	# Move binary into place
	mv "$tmp_dir/tpm-go" "$root_dir/tpm-go" 2>/dev/null || { rm -rf "$tmp_dir"; return; }
	chmod +x "$root_dir/tpm-go" 2>/dev/null
	rm -rf "$tmp_dir"

	echo "$root_dir/tpm-go"
}
