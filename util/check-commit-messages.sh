#!/usr/bin/env bash

# shellcheck disable=1091
source "$(dirname "${BASH_SOURCE[0]}")"/common.bash || exit

usage() (
	cat <<-...
	Usage: $0 <commit-range>|<file>
	  <commit-range>: A range of commits in the form hash..hash
	  <file>: A file containing a list of commit hashes, one per line"
	...
)

main() (
	require git head sed

	case $# in
		0) usage; exit ;;
		1) range_or_file=$1 ;;
		*) die \
				'Error: Too many arguments.' \
				'' \
				"$(usage)" ;;
	esac

	# Determine input type
	range_or_file=${1:-HEAD}
	if [[ -f $range_or_file ]]; then
		message=$(< "$range_or_file")
		validate-commit-message "$range_or_file" "$message" ||
			die "Commit message in $range_or_file is invalid."

	else
		ok=true
		commits=$(git rev-list "$range_or_file")
		for commit in $commits; do
			message=$(git log --format=%B -n 1 "$commit")
			validate-commit-message "$commit" "$message" ||
				ok=false
		done
		$ok || die 'At least one commit message is invalid.'
	fi
)

validate-commit-message() (
	commit_or_file=$1
	message=$2
	subject=$(head -n1 <<<"$message")
	length=${#subject}
	errors=()

	declare -A lines_with_errors
	last_line_with_error=0

	[[ $subject =~ ^(feat|fix|docs|style|refactor|perf|test|chore)(\(.*\))?: ]] &&
		errors+=('do not use conventional commit format for subject on line 1')

	# subject should not start with square brackets
	[[ $subject =~ ^\[.*\] ]] &&
		errors+=('subject should not start with square brackets on line 1')

	[[ $subject =~ ^[A-Z] ]] ||
		errors+=('subject should start with a capital letter on line 1')

	[[ $subject == *. ]] &&
		errors+=('subject should not end with a period on line 1')

	[[ $subject == *'  '* ]] &&
		errors+=('subject should not contain consecutive spaces on line 1')

	[[ $subject == *' ' ]] &&
		errors+=('subject should not have trailing space(s) on line 1')

	if [[ $length -lt 20 ]]; then
		errors+=("subject should be longer than 20 characters (current: $length) on line 1")
	elif [[ $length -gt 50 ]]; then
		errors+=("subject should be shorter than 50 characters (current: $length) on line 1")
	fi

	if [[ ${#errors[*]} -gt 0 ]]; then
		lines_with_errors[1]=true
		last_line_with_error=1
	fi

	if [[ $(sed -n '2p' <<<"$message") ]]; then
		errors+=('subject and body should be separated by a single blank line on line 2')
		lines_with_errors[2]=true
		last_line_with_error=2
	fi

	body=$(sed -n '3,$p' <<<"$message")
	i=3
	while IFS= read -r line; do
		if [[ $line == *' ' ]]; then
			errors+=("body should not have trailing space(s) on line $i")
			lines_with_errors[$i]=true
			last_line_with_error=$i
		fi
		((i++))
	done <<<"$body"

	# Return if no errors:
	[[ ${#errors[@]} -eq 0 ]] && return

	# Report errors to stderr:
	(
		echo -e "${R}Error: $commit_or_file has invalid message:$Z\n"

		# read the message and add the line number in front of each line, and use
		# warn_color to display a line with an error based on line_with_errors
		i=0
		C=$Y
		while IFS= read -r line && [[ $((++i)) -lt $last_line_with_error ]]; do
			(
				[[ ${lines_with_errors[$i]:-} ]] || C=''
				echo -e "${C}Line $i: $line$Z"
			)
		done <<<"$message"

		echo
		printf -- '- %s\n' "${errors[@]}"
		echo
	) >&2

	return 1
)

main "$@"
