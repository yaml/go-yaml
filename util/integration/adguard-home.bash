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
            'make VERBOSE=1 go-test'
        check build \
            'go build ./...'
        check go-os-check \
            'make VERBOSE=1 go-os-check'
        check go-lint \
            'make VERBOSE=1 go-lint'
        pending ci-workflow-yaml \
            '.github/workflows/workflow.yaml (ESLint, TestJob, BuildSnapshot): remaining CI setup and local execution need translation'
        pending ci-build-yaml \
            '.github/workflows/build.yaml (Build, TestE2E): remaining CI setup and local execution need translation'
        pending ci-private-yaml \
            '.github/workflows/private.yaml (TestFrontend, LintAndTestJobDocker, LintAndTestJobMacOS, LintAndTestJobWindows): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
