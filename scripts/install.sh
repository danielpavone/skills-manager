#!/usr/bin/env sh
set -eu

project_name="skills-manager"
repository="${SKILLS_MANAGER_REPO:-danielpavone/skills-manager}"
version="${SKILLS_MANAGER_VERSION:-latest}"
install_dir="${SKILLS_MANAGER_INSTALL_DIR:-}"
download_base="${SKILLS_MANAGER_DOWNLOAD_BASE_URL:-}"
curl_bin="${SKILLS_MANAGER_CURL_BIN:-curl}"
test_mode="${SKILLS_MANAGER_TEST_MODE:-0}"

fail() {
  printf '%s\n' "skills-manager installer: $1" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "comando '$1' não encontrado; esperado um ambiente com '$1' disponível"
}

detect_platform() {
  os_name=$(uname -s 2>/dev/null || fail "não foi possível identificar o sistema operacional; esperado Linux ou Darwin")
  case "$os_name" in
    Linux) platform_os=Linux ;;
    Darwin) platform_os=Darwin ;;
    *) fail "sistema operacional '$os_name' não suportado; esperado Linux ou Darwin" ;;
  esac

  architecture=$(uname -m 2>/dev/null || fail "não foi possível identificar a arquitetura; esperado x86_64 ou arm64")
  case "$architecture" in
    x86_64|amd64) platform_arch=x86_64 ;;
    arm64|aarch64) platform_arch=arm64 ;;
    *) fail "arquitetura '$architecture' não suportada; esperado x86_64 ou arm64" ;;
  esac
}

resolve_download_base() {
  if [ -n "$download_base" ]; then
    return
  fi
  if [ "$version" = latest ]; then
    download_base="https://github.com/$repository/releases/latest/download"
    return
  fi
  download_base="https://github.com/$repository/releases/download/$version"
}

validate_download_base() {
  case "$download_base" in
    https://*) return ;;
    *) [ "$test_mode" = 1 ] || fail "URL-base '$download_base' não é HTTPS; esperado uma URL https:// para downloads" ;;
  esac
}

download_file() {
  source_url="$1"
  destination="$2"
  "$curl_bin" -fsSL "$source_url" -o "$destination" || fail "falha ao baixar '$source_url'; esperado um artefato acessível por HTTPS"
}

checksum_tool() {
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s\n' sha256sum
    return
  fi
  require_command shasum
  printf '%s\n' shasum
}

verify_checksum() {
  checksum_file="$1"
  artifact_name="$2"
  artifact_path="$3"
  expected=$(awk -v artifact="$artifact_name" '$2 == artifact { print $1; exit }' "$checksum_file")
  [ -n "$expected" ] || fail "checksum de '$artifact_name' ausente em '$checksum_file'; esperado SHA-256 publicado"
  [ "${#expected}" -eq 64 ] || fail "checksum '$expected' para '$artifact_name' tem formato inválido; esperado 64 caracteres hexadecimais"
  case "$expected" in
    *[!0123456789abcdefABCDEF]*) fail "checksum '$expected' para '$artifact_name' tem formato inválido; esperado 64 caracteres hexadecimais" ;;
  esac

  tool=$(checksum_tool)
  if [ "$tool" = sha256sum ]; then
    actual=$(sha256sum "$artifact_path" | awk '{ print $1}')
  else
    actual=$(shasum -a 256 "$artifact_path" | awk '{ print $1}')
  fi
  [ "$actual" = "$expected" ] || fail "checksum divergente para '$artifact_name': obtido '$actual', esperado '$expected'; binário existente não foi substituído"
}

choose_install_dir() {
  if [ -n "$install_dir" ]; then
    return
  fi
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir=/usr/local/bin
    return
  fi
  local_home="${HOME:-.}"
  install_dir="$local_home/.local/bin"
}

install_archive() {
  archive="$1"
  target="$install_dir/$project_name"
  staging_dir=$(mktemp -d "${TMPDIR:-/tmp}/skills-manager.XXXXXX") || fail "não foi possível criar diretório temporário; esperado um diretório gravável"
  trap 'rm -rf "$download_dir" "$staging_dir"' EXIT HUP INT TERM
  tar -xzf "$archive" -C "$staging_dir" || fail "não foi possível extrair '$archive'; esperado um arquivo tar.gz válido"
  binary="$staging_dir/$project_name"
  [ -f "$binary" ] || fail "arquivo '$project_name' não encontrado em '$archive'; esperado o binário na raiz do arquivo"
  mkdir -p "$install_dir" || fail "não foi possível criar '$install_dir'; esperado um diretório de instalação gravável"
  temporary_target="$install_dir/.${project_name}.new.$$"
  install -m 0755 "$binary" "$temporary_target" || fail "não foi possível preparar '$temporary_target'; esperado um destino gravável"
  mv -f "$temporary_target" "$target" || fail "não foi possível instalar '$target'; esperado um diretório de instalação gravável"
  rm -rf "$staging_dir"
  printf 'skills-manager instalado em %s\n' "$target"
}

require_command uname
require_command awk
require_command tar
require_command mktemp
require_command install
require_command mv
detect_platform
resolve_download_base
validate_download_base
choose_install_dir

asset_name="${project_name}_${platform_os}_${platform_arch}"
case "$platform_os" in
  Linux|Darwin) archive_name="$asset_name.tar.gz" ;;
esac
checksum_name="${project_name}_checksums.txt"
download_dir=$(mktemp -d "${TMPDIR:-/tmp}/skills-manager-download.XXXXXX") || fail "não foi possível criar diretório temporário; esperado um diretório gravável"
trap 'rm -rf "$download_dir"' EXIT HUP INT TERM
archive_path="$download_dir/$archive_name"
checksum_path="$download_dir/$checksum_name"
download_file "$download_base/$checksum_name" "$checksum_path"
download_file "$download_base/$archive_name" "$archive_path"
verify_checksum "$checksum_path" "$archive_name" "$archive_path"
install_archive "$archive_path"
rm -rf "$download_dir"
