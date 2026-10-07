#!/bin/sh
# Installs the codefall binary from install.codefall.dev.
#
#   curl -fsSL https://install.codefall.dev/sh | sh
#   curl -fsSL https://install.codefall.dev/sh | sh -s -- 0.28.0
#
# The release workflow uploads this file to the bucket root as `sh`. With no argument it installs the
# version named in `latest`; an argument pins a version, with or without a leading `v`.
# CODEFALL_INSTALL_DIR sets where the binary goes (default: ~/.local/bin).
#
# After installing, it writes a receipt to $XDG_STATE_HOME/codefall/install.json (default:
# ~/.local/state/codefall/install.json) naming the binary's path and version. `codefall update` reads
# it to know the binary came from this script (ADR-015).

set -eu

BASE_URL="https://install.codefall.dev"

say() { printf 'codefall: %s\n' "$*"; }
fail() { printf 'codefall: %s\n' "$*" >&2; exit 1; }

fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -q "$1" -O "$2"
	else
		fail "needs curl or wget"
	fi
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d' ' -f1
	else
		fail "needs sha256sum or shasum to verify the download"
	fi
}

# json_string prints its argument as a JSON string, escaping backslashes and double quotes.
json_string() {
	printf '"%s"' "$(printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g')"
}

# write_receipt records the installed binary's path, with symbolic links in its directory resolved,
# and its version. A receipt that cannot be written leaves the install in place and says so.
write_receipt() {
	state="${XDG_STATE_HOME:-$HOME/.local/state}/codefall"
	path="$(cd "$(dirname "$1")" && pwd -P)/$(basename "$1")"
	if mkdir -p "$state" &&
		printf '{\n  "path": %s,\n  "version": %s\n}\n' "$(json_string "$path")" "$(json_string "$2")" >"$state/install.json.tmp" &&
		mv "$state/install.json.tmp" "$state/install.json"; then
		return
	fi
	rm -f "$state/install.json.tmp" 2>/dev/null || true
	say "could not write $state/install.json; codefall update will not recognise this install"
}

main() {
	case "$(uname -s)" in
		Darwin) os=darwin ;;
		Linux) os=linux ;;
		*) fail "no build for $(uname -s); builds exist for macOS and Linux" ;;
	esac
	case "$(uname -m)" in
		x86_64 | amd64) arch=amd64 ;;
		arm64 | aarch64) arch=arm64 ;;
		*) fail "no build for $(uname -m); builds exist for amd64 and arm64" ;;
	esac
	# A shell running under Rosetta reports x86_64 on Apple silicon, where the arm64 build is native.
	if [ "$os" = darwin ] && [ "$arch" = amd64 ] && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null)" = 1 ]; then
		arch=arm64
	fi

	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT

	if [ $# -gt 0 ]; then
		version="${1#v}"
	else
		fetch "$BASE_URL/latest" "$tmp/latest" || fail "could not read $BASE_URL/latest"
		version="$(tr -d '[:space:]' <"$tmp/latest")"
	fi
	[ -n "$version" ] || fail "no version to install"

	archive="codefall_${version}_${os}_${arch}.tar.gz"
	say "installing $version for $os/$arch"
	fetch "$BASE_URL/$version/$archive" "$tmp/$archive" || fail "could not download $BASE_URL/$version/$archive"
	fetch "$BASE_URL/$version/checksums.txt" "$tmp/checksums.txt" || fail "could not download checksums for $version"

	want="$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")"
	[ -n "$want" ] || fail "checksums.txt has no entry for $archive"
	[ "$(sha256 "$tmp/$archive")" = "$want" ] || fail "checksum mismatch for $archive"

	tar -xzf "$tmp/$archive" -C "$tmp" codefall

	dir="${CODEFALL_INSTALL_DIR:-$HOME/.local/bin}"
	mkdir -p "$dir"
	mv "$tmp/codefall" "$dir/codefall"
	chmod 755 "$dir/codefall"
	say "installed $dir/codefall"
	write_receipt "$dir/codefall" "$version"

	case ":$PATH:" in
		*":$dir:"*) ;;
		*) say "$dir is not on your PATH; add it to run codefall by name" ;;
	esac
}

# The whole script is read before main runs, so a download cut short cannot run half of it.
main "$@"
