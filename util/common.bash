#! bash

# Strict settings:
# -u works poorly with arrays until Bash 4.4
shopt -s compat43 2>/dev/null ||
	{ echo "Bash 4.4 required"; exit 1; }

set -euo pipefail
shopt -s inherit_errexit

# ANSI color code vars:
R="\033[31m"
Y="\033[33m"
Z="\033[0m"

require() (
	for cmd; do
		command -v "$cmd" >/dev/null ||
			die "Error: $cmd is not installed or available in the PATH."
	done
)

# General error function:
die() {
	echo -e "$R$1$Z" >&2
	shift

	for line; do
		echo -e "$line"
	done >&2

	exit 1
}
