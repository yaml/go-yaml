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
            'go test -count=1 ./...'
        check build \
            'make build-service'
        check format \
            'make check-fmt'
        check toolchain \
            'make check-go-toolchain'
        pending ci-test-yml \
            '.github/workflows/test.yml (test, e2e-github, e2e-gitlab): remaining CI setup and local execution need translation'
        pending ci-go-lint-yml \
            '.github/workflows/go-lint.yml (golangci-lint): remaining CI setup and local execution need translation'
        pending ci-markdown-lint-yml \
            '.github/workflows/markdown-lint.yml (markdown-toc): remaining CI setup and local execution need translation'
        pending ci-codeql-yml \
            '.github/workflows/codeql.yml (analyze): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
