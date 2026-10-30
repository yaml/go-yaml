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
        check short \
            'make test-short'
        check go-tests \
            'make test'
        check build \
            'make build'
        pending ci-go-yaml \
            '.github/workflows/go.yaml (build): remaining CI setup and local execution need translation'
        pending ci-govulncheck-yml \
            '.github/workflows/govulncheck.yml (govulncheck): remaining CI setup and local execution need translation'
        pending ci-codeql-analysis-yml \
            '.github/workflows/codeql-analysis.yml (analyse): remaining CI setup and local execution need translation'
        pending ci-spelling-yml \
            '.github/workflows/spelling.yml (spelling): remaining CI setup and local execution need translation'
        pending ci-typos-yaml \
            '.github/workflows/typos.yaml (typos): remaining CI setup and local execution need translation'
        pending ci-zizmor-yaml \
            '.github/workflows/zizmor.yaml (zizmor): remaining CI setup and local execution need translation'
        pending ci-updatecli_test-yaml \
            '.github/workflows/updatecli_test.yaml (updatecli): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
