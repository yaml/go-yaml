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
        pending ci-checks-yml \
            '.github/workflows/checks.yml (ensure_snapshots_are_being_cleaned, filenames, format, tidy, lint): remaining CI setup and local execution need translation'
        pending ci-tests-yml \
            '.github/workflows/tests.yml (prepare_test_image_testdata, tests, docker): remaining CI setup and local execution need translation'
        pending ci-codeql-analysis-yml \
            '.github/workflows/codeql-analysis.yml (analyze): remaining CI setup and local execution need translation'
        pending ci-renovate-validator-yml \
            '.github/workflows/renovate-validator.yml (validate): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
