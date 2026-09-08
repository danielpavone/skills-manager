#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/skills-manager-installer-test.XXXXXX")
trap 'rm -rf "$test_root"' EXIT HUP INT TERM
fixture_dir="$test_root/release"
install_dir="$test_root/bin"
archive_root="$test_root/archive"
mkdir -p "$fixture_dir" "$archive_root"
platform_os=$(uname -s)
case "$platform_os" in
  Linux) platform_os=linux ;;
  Darwin) platform_os=darwin ;;
  *) printf '%s\n' "teste do instalador: SO '$platform_os' não suportado" >&2; exit 1 ;;
esac
platform_arch=$(uname -m)
case "$platform_arch" in
  x86_64|amd64) platform_arch=amd64 ;;
  arm64|aarch64) platform_arch=arm64 ;;
  *) printf '%s\n' "teste do instalador: arquitetura '$platform_arch' não suportada" >&2; exit 1 ;;
esac
archive_name="skills-manager_${platform_os}_${platform_arch}.tar.gz"
printf '%s\n' '#!/bin/sh' 'printf "fixture\\n"' > "$archive_root/skills-manager"
chmod 0755 "$archive_root/skills-manager"
tar -czf "$fixture_dir/$archive_name" -C "$archive_root" skills-manager
if command -v sha256sum >/dev/null 2>&1; then
  sha256=$(sha256sum "$fixture_dir/$archive_name" | awk '{ print $1 }')
else
  sha256=$(shasum -a 256 "$fixture_dir/$archive_name" | awk '{ print $1 }')
fi
printf '%s  %s\n' "$sha256" "$archive_name" > "$fixture_dir/skills-manager_checksums.txt"
mkdir -p "$install_dir"
printf '%s\n' 'binário anterior' > "$install_dir/skills-manager"

printf '%s  %s\n' '0000000000000000000000000000000000000000000000000000000000000000' "$archive_name" > "$fixture_dir/skills-manager_checksums.txt"
set +e
SKILLS_MANAGER_TEST_MODE=1 \
SKILLS_MANAGER_DOWNLOAD_BASE_URL=fixture://release \
SKILLS_MANAGER_CURL_BIN="$PWD/scripts/test-support/fake-curl.sh" \
SKILLS_MANAGER_FIXTURE_DIR="$fixture_dir" \
SKILLS_MANAGER_INSTALL_DIR="$install_dir" \
SKILLS_MANAGER_VERSION=v0.1.0 \
sh scripts/install.sh >/dev/null 2>&1
status=$?
set -e
[ "$status" -ne 0 ] || { printf '%s\n' 'teste do instalador: checksum divergente deveria falhar' >&2; exit 1; }
[ "$(cat "$install_dir/skills-manager")" = 'binário anterior' ] || { printf '%s\n' 'teste do instalador: checksum divergente substituiu o binário' >&2; exit 1; }

printf '%s  %s\n' "$sha256" "$archive_name" > "$fixture_dir/skills-manager_checksums.txt"

SKILLS_MANAGER_TEST_MODE=1 \
SKILLS_MANAGER_DOWNLOAD_BASE_URL=fixture://release \
SKILLS_MANAGER_CURL_BIN="$script_dir/test-support/fake-curl.sh" \
SKILLS_MANAGER_FIXTURE_DIR="$fixture_dir" \
SKILLS_MANAGER_INSTALL_DIR="$install_dir" \
SKILLS_MANAGER_VERSION=v0.1.0 \
sh "$script_dir/install.sh"

[ -x "$install_dir/skills-manager" ] || { printf '%s\n' 'teste do instalador: binário não instalado' >&2; exit 1; }
[ "$("$install_dir/skills-manager")" = 'fixture' ] || { printf '%s\n' 'teste do instalador: binário instalado não executou' >&2; exit 1; }
printf '%s\n' 'instalador POSIX validado com fixtures locais'
