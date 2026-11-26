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
        check build \
            'make build'
        check modules \
            'make mod/check'
        check license \
            'make license/check'
        pending ci-ci-experimental-yml \
            '.github/workflows/ci-experimental.yml (build_test): remaining CI setup and local execution need translation'
        pending ci-ci-sync-preview-yml \
            '.github/workflows/ci-sync-preview.yml (build_test, release_dry_run): remaining CI setup and local execution need translation'
        pending ci-ci-sync-deployment-yml \
            '.github/workflows/ci-sync-deployment.yml (build_test, release_dry_run): remaining CI setup and local execution need translation'
        pending ci-interop-tests-yml \
            '.github/workflows/interop-tests.yml (interop-tests): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
