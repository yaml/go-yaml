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
        pending ci-ci-yml \
            '.github/workflows/ci.yml (validate, binary, bin-image-test, relay-image-test, test, e2e, coverage): remaining CI setup and local execution need translation'
        pending ci-docs-upstream-yml \
            '.github/workflows/docs-upstream.yml (docs-yaml, validate): remaining CI setup and local execution need translation'
        pending ci-scorecards-yml \
            '.github/workflows/scorecards.yml (analysis): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
