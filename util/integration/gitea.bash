#!/usr/bin/env bash

# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

# shellcheck source=util/integration/common.bash
source "${INTEGRATION_ROOT:?}/util/integration/common.bash"

export CC=gcc CXX=g++

makes gcc.mk GCC

case ${1:?Expected prepare or test} in
    prepare)
        export CC=gcc CXX=g++
        make deps-backend generate-go
        ;;
    test)
        check go-tests \
            'make test-backend'
        check build \
            'TAGS=bindata make backend'
        check checks-backend \
            'make --always-make checks-backend'
        pending ci-pull-db-tests-yml \
            '.github/workflows/pull-db-tests.yml (test-pgsql-shard-1, test-pgsql-shard-2, test-sqlite, test-unit, test-mysql, test-mssql): remaining CI setup and local execution need translation'
        pending ci-pull-e2e-tests-yml \
            '.github/workflows/pull-e2e-tests.yml (test-e2e): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
