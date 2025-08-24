#!/usr/bin/env bash

# -u works poorly with arrays until Bash 4.4
shopt -s compat43 2>/dev/null ||
	{ echo "Bash 4.4 required"; exit 1; }
set -euo pipefail
shopt -s inherit_errexit

usage() {
	echo "\
Usage: $0 <commit-range|file>
  <commit-range>: A range of commits in the form hash..hash
  <file>: A file containing a list of commit hashes, one per line"
	exit 1
}

reset_color="\033[0m"
error_color="\033[31m"
warning_color="\033[33m"

validate_commit_message() {
	local commit_or_file=$1
	local message=$2
	local subject
	subject=$(echo "$message" | head -n 1)
	local length=${#subject}
	local errors=()

	declare -A lines_with_errors
	local subject_has_error=false
	local last_line_with_error=0

	if [[ $subject =~ ^(feat|fix|docs|style|refactor|perf|test|chore)(\(.*\))?: ]]; then
		errors+=('do not use conventional commit format for subject on line 1')
		subject_has_error=true
	fi

	# subject should not start with square brackets
	if [[ $subject =~ ^\[.*\] ]]; then
		errors+=('subject should not start with square brackets on line 1')
		subject_has_error=true
	fi

	if [[ ! $subject =~ ^[A-Z] ]]; then
		errors+=('subject should start with a capital letter on line 1')
		subject_has_error=true
	fi

	if [[ $subject == *. ]]; then
		errors+=('subject should not end with a period on line 1')
		subject_has_error=true
	fi

	if [[ $subject == *'  '* ]]; then
		errors+=('subject should not contain consecutive spaces on line 1')
		subject_has_error=true
	fi

	if [[ $subject == *' ' ]]; then
		errors+=('subject should not have trailing space(s) on line 1')
		subject_has_error=true
	fi

	if [[ $length -lt 20 ]]; then
		errors+=("subject should be longer than 20 characters (current: $length) on line 1")
		subject_has_error=true
	elif [[ $length -gt 50 ]]; then
		errors+=("subject should be shorter than 50 characters (current: $length) on line 1")
		subject_has_error=true
	fi

	if $subject_has_error; then
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

	if [[ ${#errors[@]} -gt 0 ]]; then
		echo -e "${error_color}Error: $commit_or_file has invalid message:${reset_color}"
		echo
		# read the message and add the line number in front of each line, and use warn_color to display a line with an error based on line_with_errors

		local i=0
		while IFS= read -r line; do
			((i++))
			local line_color
			if [[ -n ${lines_with_errors[$i]:-} ]]; then
				line_color=$warning_color
			fi
			echo -e "${line_color}Line $i: $line${reset_color}"
			if [[ $i -ge $((last_line_with_error)) ]]; then
				break
			fi
		done <<<"$message"
		echo
		printf -- "- %s\n" "${errors[@]}"
		echo
		return 1
	fi
	return 0
}

for cmd in git sed head; do
	if ! command -v "$cmd" >/dev/null 2>&1; then
		echo "Error: $cmd is not installed or available in the PATH."
		exit 1
	fi
done

case $# in
0) usage ;;
1) range_or_file=$1 ;;
*)
	echo 'Error: Too many arguments.'
	usage
	;;
esac

# Determine input type
range_or_file=${1:-HEAD}
if [[ -f $range_or_file ]]; then
	message=$(cat "$range_or_file")
	if ! validate_commit_message "$range_or_file" "$message"; then
		echo -e "${error_color}Commit message in $range_or_file is invalid.${reset_color}"
		exit 1
	fi
else
	fail=0
	commits=$(git rev-list "$range_or_file")
	for commit in $commits; do
		message=$(git log --format=%B -n 1 "$commit")
		if ! validate_commit_message "$commit" "$message"; then
			fail=1
		fi
	done
	if [[ $fail -eq 1 ]]; then
		echo -e "${error_color}At least one commit message is invalid.${reset_color}"
		exit 1
	fi
fi
