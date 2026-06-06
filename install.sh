#!/bin/sh
set -eu

REPO="${TAILEDBOX_REPO:-f1forhelp/tailed-box-cli}"
VERSION="${TAILEDBOX_VERSION:-v0.1.0}"

case "$VERSION" in
	v*)
		RELEASE_TAG="$VERSION"
		VERSION_NUMBER="${VERSION#v}"
		;;
	*)
		RELEASE_TAG="v${VERSION}"
		VERSION_NUMBER="$VERSION"
		;;
esac

BASE_URL="https://github.com/${REPO}/releases/download/${RELEASE_TAG}"

log() {
	printf '%s\n' "$*"
}

error() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

need_command() {
	command -v "$1" >/dev/null 2>&1 || error "required command not found: $1"
}

download() {
	url="$1"
	output="$2"

	if command -v curl >/dev/null 2>&1; then
		curl -fL --retry 3 -o "$output" "$url"
	elif command -v wget >/dev/null 2>&1; then
		wget -O "$output" "$url"
	else
		error "curl or wget is required to download release assets"
	fi
}

run_as_root() {
	if [ "$(id -u)" -eq 0 ]; then
		"$@"
	elif command -v sudo >/dev/null 2>&1; then
		sudo "$@"
	else
		error "root privileges are required; rerun as root or install sudo"
	fi
}

detect_target() {
	kernel="$(uname -s)"
	machine="$(uname -m)"

	case "$kernel" in
		Linux)
			;;
		*)
			error "OS not supported: ${kernel}. No Tailedbox build exists for this OS in ${RELEASE_TAG}."
			;;
	esac

	if [ ! -r /etc/os-release ]; then
		error "OS not supported: cannot read /etc/os-release"
	fi

	# shellcheck disable=SC1091
	. /etc/os-release
	os_id="${ID:-unknown}"

	case "$os_id" in
		debian)
			;;
		*)
			error "OS not supported: ${os_id}. ${RELEASE_TAG} currently provides Debian builds only."
			;;
	esac

	case "$machine" in
		x86_64|amd64)
			arch="amd64"
			;;
		*)
			error "architecture not supported: ${machine}. No Debian build exists for this architecture in ${RELEASE_TAG}."
			;;
	esac

	printf 'debian:%s\n' "$arch"
}

verify_checksum() {
	checksum_file="$1"
	asset_name="$2"
	package_file="$3"

	need_command sha256sum
	need_command awk

	expected="$(awk -v asset="$asset_name" '{ name = $2; sub(/^.*\//, "", name); if (name == asset) { print $1; exit } }' "$checksum_file")"
	if [ -z "$expected" ]; then
		error "checksum for ${asset_name} was not found in checksums.txt"
	fi

	actual="$(sha256sum "$package_file" | awk '{ print $1 }')"
	if [ "$actual" != "$expected" ]; then
		error "checksum mismatch for ${asset_name}"
	fi
}

install_debian_package() {
	package_file="$1"

	if command -v apt-get >/dev/null 2>&1; then
		run_as_root apt-get install -y "$package_file"
	elif command -v dpkg >/dev/null 2>&1; then
		run_as_root dpkg -i "$package_file"
	else
		error "apt-get or dpkg is required to install the Debian package"
	fi
}

main() {
	target="$(detect_target)"

	case "$target" in
		debian:amd64)
			asset="tailedbox_${VERSION_NUMBER}_amd64.deb"
			;;
		*)
			error "OS not supported: no release asset mapping for ${target}"
			;;
	esac

	tmpdir="$(mktemp -d)"
	trap 'rm -rf "$tmpdir"' EXIT INT TERM

	package_file="${tmpdir}/${asset}"
	checksum_file="${tmpdir}/checksums.txt"

	log "Detected supported target: ${target}"
	log "Downloading ${asset} from ${REPO} ${RELEASE_TAG}..."
	download "${BASE_URL}/${asset}" "$package_file"
	download "${BASE_URL}/checksums.txt" "$checksum_file"

	log "Verifying checksum..."
	verify_checksum "$checksum_file" "$asset" "$package_file"

	log "Installing ${asset}..."
	install_debian_package "$package_file"

	hash -r 2>/dev/null || true
	if ! command -v tailedbox >/dev/null 2>&1; then
		error "tailedbox was installed, but it is not available on PATH. Check that /usr/bin is in your PATH."
	fi

	log "Tailedbox installed successfully."
	tailedbox version
	if tailedbox uninstall --help 2>/dev/null | grep -q -- '--all'; then
		log "To fully remove Tailedbox later, run: sudo tailedbox uninstall --all --confirm-delete DELETE"
	else
		log "To remove this release later, run: sudo tailedbox uninstall --systemd --confirm-delete DELETE && sudo apt-get purge -y tailedbox"
	fi
}

main "$@"
