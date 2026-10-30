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
            'go build ./...'
        pending ci-build-yaml \
            '.github/workflows/build.yaml (build): remaining CI setup and local execution need translation'
        pending ci-test-yaml \
            '.github/workflows/test.yaml (test): remaining CI setup and local execution need translation'
        pending ci-e2e-yaml \
            '.github/workflows/e2e.yaml (e2e): remaining CI setup and local execution need translation'
        pending ci-kind-e2e-yaml \
            '.github/workflows/kind-e2e.yaml (e2e-tests): remaining CI setup and local execution need translation'
        pending ci-modules-integration-test-yaml \
            '.github/workflows/modules-integration-test.yaml (test): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
