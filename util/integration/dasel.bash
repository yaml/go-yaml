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
            'go test -count=1 -race ./...'
        check build \
            'go build ./cmd/dasel'
        pending ci-test-yaml \
            '.github/workflows/test.yaml (test): remaining CI setup and local execution need translation'
        pending ci-golangci-lint-yaml \
            '.github/workflows/golangci-lint.yaml (golangci): remaining CI setup and local execution need translation'
        pending ci-codeql-analysis-yml \
            '.github/workflows/codeql-analysis.yml (analyze): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
