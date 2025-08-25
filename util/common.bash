#!/bash

# Strict settings:
# -u works poorly with arrays until Bash 4.4
shopt -s compat43 2>/dev/null ||
	{ echo "Bash 4.4 required"; exit 1; }

set -euo pipefail
shopt -s inherit_errexit

# ANSI color code vars:
RED="\e[1;31m"
RESET="\e[0m"

require() (
	for cmd; do
		command -v "$cmd" >/dev/null ||
			die "Error: $cmd is not installed or available in the PATH."
	done
)

# General error function:
die() {
	echo -e "$RED$1$RESET" >&2
	shift

	for line; do
		echo -e "$line"
	done >&2

	exit 1
}
