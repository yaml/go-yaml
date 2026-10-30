#!/usr/bin/env bash

# Copyright 2026 The go-yaml Project Contributors
# SPDX-License-Identifier: Apache-2.0

# shellcheck source=util/integration/common.bash
source "${INTEGRATION_ROOT:?}/util/integration/common.bash"

unset GOCCY

case ${1:?Expected prepare or test} in
    prepare)
        patch-tests
        ;;
    test)
        check go-tests \
            "go test -count=1 ${INTEGRATION_GO_FLAGS:-} ./..."
        check module-packaging \
            "go test -count=1 ${INTEGRATION_GO_FLAGS:-} -tags goinstall -run '^TestGoInstallCompatibility$' ."
        check build \
            'go build -o yq .'
        check acceptance \
            'go build -o yq .; bash scripts/acceptance.sh'
        pending ci-go-yml \
            '.github/workflows/go.yml (verify-action-digest, build): remaining CI setup and local execution need translation'
        pending ci-codeql-yml \
            '.github/workflows/codeql.yml (analyze): remaining CI setup and local execution need translation'
        pending ci-scorecard-yml \
            '.github/workflows/scorecard.yml (analysis): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
