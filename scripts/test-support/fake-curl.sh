#!/usr/bin/env sh
set -eu

fixture_dir="${SKILLS_MANAGER_FIXTURE_DIR:?SKILLS_MANAGER_FIXTURE_DIR is required}"
source_url=
destination=

while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) destination="$2"; shift 2 ;;
    -*) shift ;;
    *) source_url="$1"; shift ;;
  esac
done

[ -n "$source_url" ] || { printf '%s\n' 'fake-curl: URL ausente' >&2; exit 2; }
[ -n "$destination" ] || { printf '%s\n' 'fake-curl: destino ausente' >&2; exit 2; }
source_path="$fixture_dir/$(basename "$source_url")"
[ -f "$source_path" ] || { printf '%s\n' "fake-curl: fixture ausente '$source_path'" >&2; exit 1; }
cp "$source_path" "$destination"
