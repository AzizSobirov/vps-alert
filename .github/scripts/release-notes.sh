#!/bin/sh
# Prints the CHANGELOG.md section for a version tag, e.g. v0.4.0, and fails
# if there is none, so a release cannot go out without its notes.
set -eu
version=${1#v}
notes=$(awk -v ver="$version" '
	/^## \[/ { if (found) exit; if (index($0, "## [" ver "]") == 1) { found = 1; next } }
	/^\[[^]]+\]: / { if (found) exit }
	found { print }
' CHANGELOG.md)
if [ -z "$(printf '%s' "$notes" | tr -d '[:space:]')" ]; then
	echo "CHANGELOG.md has no section for $1" >&2
	exit 1
fi
printf '%s\n' "$notes"
