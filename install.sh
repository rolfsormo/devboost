#!/bin/sh
# devboost bootstrap installer — modeled on rustup's rustup-init.sh: a
# small, pure-POSIX-shell dispatcher whose only job is to detect OS/arch,
# fetch the matching prebuilt devboost binary, chmod it executable, and
# exec it. No application logic lives here — everything real lives in
# the binary, which is what makes this entry point small enough to
# actually read in full before running it.
#
# This is deliberately NOT wired to a real release pipeline yet — per
# the v2 architecture decision, cross-compilation/distribution stays
# local-only until it's actually needed (no GitHub Actions spend at this
# stage). DEVBOOST_INSTALL_BASE_URL lets this script be pointed at a
# local build output for testing (see install_test.sh), or at a real
# release URL once one exists.

set -eu

DEVBOOST_INSTALL_BASE_URL="${DEVBOOST_INSTALL_BASE_URL:-https://github.com/rolfsormo/devboost/releases/latest/download}"

say() {
    printf '%s\n' "devboost-install: $1"
}

err() {
    say "$1" >&2
    exit 1
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        err "need '$1' (command not found)"
    fi
}

# get_architecture ports the essential logic of rustup-init.sh's own
# architecture detection, including the Rosetta 2 trap: under Rosetta,
# 'uname -m' on Apple Silicon reports x86_64 even though the real
# hardware is arm64, so an extra sysctl check is required on Darwin.
get_architecture() {
    _ostype="$(uname -s)"
    _cputype="$(uname -m)"

    if [ "$_ostype" = Darwin ] && [ "$_cputype" = x86_64 ]; then
        if sysctl hw.optional.arm64 2>/dev/null | grep -q ': 1'; then
            _cputype=arm64
        fi
    fi

    case "$_ostype" in
        Linux)
            _ostype=linux
            ;;
        Darwin)
            _ostype=darwin
            ;;
        MINGW* | MSYS* | CYGWIN*)
            _ostype=windows
            ;;
        *)
            err "unrecognized OS type: $_ostype"
            ;;
    esac

    case "$_cputype" in
        x86_64 | amd64)
            _cputype=amd64
            ;;
        aarch64 | arm64)
            _cputype=arm64
            ;;
        *)
            err "unrecognized CPU type: $_cputype"
            ;;
    esac

    echo "${_ostype}-${_cputype}"
}

downloader() {
    _url="$1"
    _dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl --fail --silent --show-error --location "$_url" --output "$_dest"
    elif command -v wget >/dev/null 2>&1; then
        wget --quiet "$_url" -O "$_dest"
    else
        err "need 'curl' or 'wget' (neither found)"
    fi
}

main() {
    need_cmd uname
    need_cmd mktemp
    need_cmd chmod

    _platform="$(get_architecture)"
    _ext=""
    case "$_platform" in
        windows-*) _ext=".exe" ;;
    esac

    _binary_name="devboost-${_platform}${_ext}"
    _url="${DEVBOOST_INSTALL_BASE_URL}/${_binary_name}"

    say "detected platform: $_platform"
    say "downloading: $_url"

    _tmpdir="$(mktemp -d)"
    _dest="${_tmpdir}/devboost${_ext}"

    downloader "$_url" "$_dest"

    if [ ! -s "$_dest" ]; then
        err "download failed or produced an empty file: $_url"
    fi

    chmod u+x "$_dest"

    # Verify the downloaded binary is actually executable before handing
    # off — catches a noexec-mounted temp dir early rather than a
    # confusing "permission denied" from exec below (same check
    # rustup-init.sh performs).
    if [ ! -x "$_dest" ]; then
        err "downloaded binary is not executable: $_dest (is \$TMPDIR mounted noexec?)"
    fi

    exec "$_dest" "$@"
}

main "$@"
