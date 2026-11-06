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
        pending ci-build-loki-binary-yml \
            '.github/workflows/build-loki-binary.yml (build): remaining CI setup and local execution need translation'
        pending ci-check-yml \
            '.github/workflows/check.yml (check): remaining CI setup and local execution need translation'
        pending ci-govulncheck-yml \
            '.github/workflows/govulncheck.yml (govulncheck): remaining CI setup and local execution need translation'
        pending ci-helm-ci-yml \
            '.github/workflows/helm-ci.yml (call-test, list-integration-tests, run-integration-tests): remaining CI setup and local execution need translation'
        pending ci-helm-test-build-yml \
            '.github/workflows/helm-test-build.yml (build): remaining CI setup and local execution need translation'
        pending ci-lint-jsonnet-yml \
            '.github/workflows/lint-jsonnet.yml (check-mixin): remaining CI setup and local execution need translation'
        pending ci-nix-ci-yaml \
            '.github/workflows/nix-ci.yaml (lint, packages): remaining CI setup and local execution need translation'
        pending ci-operator-yaml \
            '.github/workflows/operator.yaml (docs, lint, build-manager, build-broker, test): remaining CI setup and local execution need translation'
        pending ci-secret-scanning-yml \
            '.github/workflows/secret-scanning.yml (TruffleHog): remaining CI setup and local execution need translation'
        pending ci-snyk-yml \
            '.github/workflows/snyk.yml (snyk-scan-ci): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
