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
        pending ci-ci-yaml \
            '.github/workflows/ci.yaml (version-check, build-sdk, build-rust, build-cog, test-playground, fmt-go, fmt-rust, fmt-python, check-llm-docs, check-stubs, lint-go, lint-rust, lint-rust-deny, lint-rust-advisories, lint-python, lint-docs, test-go, fuzz-go, test-rust, test-python, test-coglet-python, integration-shards, test-integration, release-dry-run): remaining CI setup and local execution need translation'
        pending ci-codeql-yml \
            '.github/workflows/codeql.yml (analyze): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
