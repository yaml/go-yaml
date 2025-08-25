# Use strict settings:
set -euo pipefail

# ANSI color code vars:
RED="\e[1;31m"
RESET="\e[0m"

require-bash-4.4+() {
	if ! shopt -s compat43 2>/dev/null; then
		local bash version
		bash=$(command -v bash)
		version=$("$bash" -c "echo \$BASH_VERSION")

		cat <<-... >&2
		Error: 'bash' version 4.4+ is required to be first in your PATH.

		You currently have:
		$bash
		$version
		...

		exit 1
	fi

	# inherit_errexit is important and introduced in 4.4
	shopt -s inherit_errexit
}

require-commands() (
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
