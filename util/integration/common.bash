# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

cd -- "${2:?Expected prepare or test and a source directory}"
integration_status=0

milliseconds() {
    local now=${EPOCHREALTIME:-}
    if [[ -n $now ]]; then
        printf '%s\n' "$(( ${now%.*} * 1000 + 10#${now#*.} / 1000 ))"
    else
        printf '%s000\n' "$(date +%s)"
    fi
}

event() {
    if [[ -n ${INTEGRATION_EVENTS_FD:-} ]]; then
        printf '%s\n' "$1" >&"$INTEGRATION_EVENTS_FD"
    fi
}

result() {
    local record
    printf -v record '%s\t%s\t%s\t%s\t%s' \
        "$1" "$2" "${3:-}" "${4:-0}" "${5:--1}"
    printf '%s\n' "$record" >> "$INTEGRATION_RESULTS"
    printf '%s: %s %s\n' "$1" "$2" "${3:-}"
    event $'end\t'"$record"
}

# Commands run in a separate strict shell so one failure cannot hide another.
check() {
    local resolved started code
    started=$(milliseconds)
    event $'start\t'"$1"
    if [[ $INTEGRATION_PHASE == candidate ]]; then
        if ! resolved=$(go list -m -f '{{.Dir}}' go.yaml.in/yaml/v4) ||
            [[ $resolved != "$INTEGRATION_ROOT" ]]; then
            result "$1" setup-error \
                'Candidate does not resolve to this checkout' \
                "$(( $(milliseconds) - started ))"
            integration_status=1
            return
        fi
    fi
    if bash -euo pipefail -c "$2"; then
        result "$1" pass '' "$(( $(milliseconds) - started ))" 0
    else
        code=$?
        result "$1" fail "Command exited with status $code" \
            "$(( $(milliseconds) - started ))" "$code"
        integration_status=1
    fi
}

skip() {
    result "$1" skip "$2"
}

pending() {
    result "$1" setup-needed "$2"
}

# Use this only for host capabilities, not installable tools.
check-if() {
    if bash -euo pipefail -c "$2"; then
        check "$1" "$3"
    else
        skip "$1" "Host capability unavailable: $2"
    fi
}

# Install a tool through Makes, then import the module's exported environment.
# An empty module selects an existing rule in the project's GNUmakefile.
makes() {
    local module=$1 variable=$2 entry
    shift 2
    make --no-print-directory -s -C "$INTEGRATION_ROOT" \
        -f GNUmakefile -f util/integration/tools.mk \
        INTEGRATION_TOOL_MODULE="$module" \
        INTEGRATION_TOOL_VARIABLE="$variable" \
        INTEGRATION_TOOL_ENV="$INTEGRATION_LOG_DIR/tools.env" \
        "$@" integration-tool
    while IFS= read -r -d '' entry; do
        case ${entry%%=*} in
            PATH|GOROOT|GOPATH|GOMODCACHE|GOCACHE|CARGO_HOME|RUSTUP_HOME|\
            UV_PYTHON_INSTALL_DIR|VIRTUAL_ENV)
                export "${entry?}"
                ;;
        esac
    done < "$INTEGRATION_LOG_DIR/tools.env"
}

patch-tests() {
    local patch
    for patch in "$INTEGRATION_ROOT/testdata/integration/$INTEGRATION_PROJECT"/*.patch; do
        [[ -e $patch ]] || continue
        echo "Applying accommodation: ${patch##*/}"
        git apply "$patch"
    done
}

finish() {
    touch "$INTEGRATION_COMPLETION"
    exit "$integration_status"
}
