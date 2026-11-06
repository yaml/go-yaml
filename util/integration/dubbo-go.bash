#!/usr/bin/env bash

# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

# shellcheck source=util/integration/common.bash
source "${INTEGRATION_ROOT:?}/util/integration/common.bash"

case ${1:?Expected prepare or test} in
    prepare)
        :
        ;;
    test)
        check go-tests \
            'make test'
        check race \
            'make test-race'
        check build \
            'go build ./...'
        check lint \
            'make lint'
        pending ci-github-actions-yml \
            '.github/workflows/github-actions.yml (license, unit-test, Integration-Test): remaining CI setup and local execution need translation'
        pending ci-tools-yml \
            '.github/workflows/tools.yml (format, lint, openapi-tests, generated-projects): remaining CI setup and local execution need translation'
        pending ci-riscv-build-yml \
            '.github/workflows/riscv-build.yml (RISC-V-Build): remaining CI setup and local execution need translation'
        pending ci-codeql-analysis-yml \
            '.github/workflows/codeql-analysis.yml (analyse): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
