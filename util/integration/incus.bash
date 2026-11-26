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
        pending ci-build-yml \
            '.github/workflows/build.yml (build): remaining CI setup and local execution need translation'
        pending ci-tests-yml \
            '.github/workflows/tests.yml (code-tests, system-tests, ovn-tests, extended-tests, vm-tests, clustering-vm-tests, client, documentation): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
