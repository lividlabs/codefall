#!/bin/sh
# Publishes one release to the install-codefall-dev Space behind install.codefall.dev, which
# scripts/install.sh reads.
#
#   scripts/publish-release.sh <version> <dir> [--no-latest]
#
# <dir> holds the release's .tar.gz archives and checksums.txt. The version folder goes up first,
# then the install script as `sh`, then `latest` last, so `latest` never names a folder that is still
# uploading. --no-latest uploads the folder alone, for republishing a release older than the current
# one. Credentials come from AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.

set -eu

[ $# -ge 2 ] || { echo "usage: $0 <version> <dir> [--no-latest]" >&2; exit 2; }
version="${1#v}"
dir="$2"
move_latest=true
[ "${3:-}" = "--no-latest" ] && move_latest=false

bucket=s3://install-codefall-dev
export AWS_DEFAULT_REGION=nyc3

s3() { aws s3 cp --endpoint-url https://nyc3.digitaloceanspaces.com --acl public-read "$@"; }

set -- "$dir"/*.tar.gz
[ -e "$1" ] || { echo "no .tar.gz archives in $dir" >&2; exit 1; }
[ -f "$dir/checksums.txt" ] || { echo "no checksums.txt in $dir" >&2; exit 1; }

# A version folder never changes once written; `sh` and `latest` change in place.
for f in "$@" "$dir/checksums.txt"; do
	s3 --cache-control "public, max-age=31536000, immutable" "$f" "$bucket/$version/$(basename "$f")"
done

$move_latest || exit 0

s3 --cache-control no-cache --content-type "text/plain; charset=utf-8" \
	"$(dirname "$0")/install.sh" "$bucket/sh"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
printf '%s\n' "$version" >"$tmp"
s3 --cache-control no-cache --content-type "text/plain; charset=utf-8" "$tmp" "$bucket/latest"
