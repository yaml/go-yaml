#!/usr/bin/env bash

# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

# shellcheck source=util/integration/common.bash
source "${INTEGRATION_ROOT:?}/util/integration/common.bash"

case ${1:?Expected prepare or test} in
    prepare)
        mkdir -p dist
        touch dist/empty
        ;;
    test)
        check go-tests \
            'make test'
        check build \
            'make build'
        check license \
            'make license'
        pending ci-default-yml \
            '.github/workflows/default.yml (build, test, lint, ui, integration-shard, integration): remaining CI setup and local execution need translation'
        pending ci-codeql-yml \
            '.github/workflows/codeql.yml (analyze): remaining CI setup and local execution need translation'
        pending ci-openapi-validate-yml \
            '.github/workflows/openapi-validate.yml (validate-openapi): remaining CI setup and local execution need translation'
        pending ci-schema-yml \
            '.github/workflows/schema.yml (build): remaining CI setup and local execution need translation'
        pending ci-cli-docs-yml \
            '.github/workflows/cli-docs.yml (clidocs): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
