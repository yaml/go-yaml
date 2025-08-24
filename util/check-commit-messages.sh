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
		validate_commit_message "$range_or_file" "$message" ||
			die "Commit message in $range_or_file is invalid."
	else
		ok=true
		commits=$(git rev-list "$range_or_file")
		for commit in $commits; do
			message=$(git log --format=%B -n 1 "$commit")
			validate_commit_message "$commit" "$message" ||
				ok=false
		done
		$ok || die 'At least one commit message is invalid.'
	fi
)

validate_commit_message() (
	commit_or_file=$1
	message=$2
	subject=$(echo "$message" | head -n 1)
	length=${#subject}
	errors=()

	declare -A lines_with_errors
	last_line_with_error=0

	if [[ $subject =~ ^(feat|fix|docs|style|refactor|perf|test|chore)(\(.*\))?: ]]; then
		errors+=('do not use conventional commit format for subject on line 1')
	fi

	# subject should not start with square brackets
	if [[ $subject =~ ^\[.*\] ]]; then
		errors+=('subject should not start with square brackets on line 1')
	fi

	if [[ ! $subject =~ ^[A-Z] ]]; then
		errors+=('subject should start with a capital letter on line 1')
	fi

	if [[ $subject == *. ]]; then
		errors+=('subject should not end with a period on line 1')
	fi

	if [[ $subject == *'  '* ]]; then
		errors+=('subject should not contain consecutive spaces on line 1')
	fi

	if [[ $subject == *' ' ]]; then
		errors+=('subject should not have trailing space(s) on line 1')
	fi

	if [[ $length -lt 20 ]]; then
		errors+=("subject should be longer than 20 characters (current: $length) on line 1")
	elif [[ $length -gt 50 ]]; then
		errors+=("subject should be shorter than 50 characters (current: $length) on line 1")
	fi

	if [[ ${#errors[*]} -gt 0 ]]; then
		lines_with_errors[1]=true
		last_line_with_error=1
	fi

	if [[ $(echo "$message" | sed -n '2p') != '' ]]; then
		errors+=('subject and body should be separated by a single blank line on line 2')
		lines_with_errors[2]=true
		last_line_with_error=2
	fi

	body=$(echo "$message" | sed -n '3,$p')
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
		echo -e "${R}Error: $commit_or_file has invalid message:$Z"
		echo

		# read the message and add the line number in front of each line, and use
		# warn_color to display a line with an error based on line_with_errors
		i=0
		while IFS= read -r line; do
			((i++))
			C=''
			[[ -n ${lines_with_errors[$i]:-} ]] && C=$Y
			echo -e "${C}Line $i: $line$Z"
			[[ $i -lt $((last_line_with_error)) ]] || break
		done <<<"$message"

		echo
		printf -- '- %s\n' "${errors[@]}"
		echo
	) >&2

	return 1
)

main "$@"
