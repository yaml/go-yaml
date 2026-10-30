#!/usr/bin/env bash

# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

# shellcheck source=util/integration/common.bash
source "${INTEGRATION_ROOT:?}/util/integration/common.bash"

makes shellcheck.mk SHELLCHECK
makes python.mk PYTHON-VENV

case ${1:?Expected prepare or test} in
    prepare)
        :
        ;;
    test)
        check go-tests \
            'go test -count=1 -race ./...'
        check build \
            'go build ./cmd/actionlint'
        check lint \
            'make lint SKIP_GO_GENERATE=true'
        pending ci-ci-yaml \
            '.github/workflows/ci.yaml (unit-tests, wasm, lint, docker): remaining CI setup and local execution need translation'
        pending ci-codeql-yaml \
            '.github/workflows/codeql.yaml (analyze-go, analyze-ts): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
