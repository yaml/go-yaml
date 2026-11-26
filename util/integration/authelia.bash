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
            'go build ./...'
        pending ci-codeql-analysis-yml \
            '.github/workflows/codeql-analysis.yml (analyze): remaining CI setup and local execution need translation'
        pending ci-scorecard-yml \
            '.github/workflows/scorecard.yml (analysis): remaining CI setup and local execution need translation'
        pending buildkite \
            '.buildkite: remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
