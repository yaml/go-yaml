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
        pending ci-golangci-lint-yml \
            '.github/workflows/golangci-lint.yml (lint, check_modules): remaining CI setup and local execution need translation'
        pending ci-docs-check-yml \
            '.github/workflows/docs-check.yml (build): remaining CI setup and local execution need translation'
        pending ci-pr-build-yml \
            '.github/workflows/pr-build.yml (build): remaining CI setup and local execution need translation'
        pending ci-container-tests-yml \
            '.github/workflows/container-tests.yml (autotag-unit-tests, container-build-and-test): remaining CI setup and local execution need translation'
        pending ci-test-all-project-types-yml \
            '.github/workflows/test-all-project-types.yml (all-project-types-pkg, all-project-types-cmd): remaining CI setup and local execution need translation'
        pending ci-test-apache-fpm-yml \
            '.github/workflows/test-apache-fpm.yml (apache-fpm): remaining CI setup and local execution need translation'
        pending ci-test-nginx-fpm-yml \
            '.github/workflows/test-nginx-fpm.yml (nginx-fpm): remaining CI setup and local execution need translation'
        pending ci-test-docker-rootless-yml \
            '.github/workflows/test-docker-rootless.yml (docker-rootless): remaining CI setup and local execution need translation'
        pending ci-test-no-bind-mounts-yml \
            '.github/workflows/test-no-bind-mounts.yml (no-bind-mounts): remaining CI setup and local execution need translation'
        pending ci-test-podman-rootless-yml \
            '.github/workflows/test-podman-rootless.yml (podman-rootless-testpkg, podman-rootless-testcmd): remaining CI setup and local execution need translation'
        pending ci-test-podman-rootless-mutagen-yml \
            '.github/workflows/test-podman-rootless-mutagen.yml (podman-rootless-mutagen-testpkg, podman-rootless-mutagen-testcmd): remaining CI setup and local execution need translation'
        pending ci-test-pull-push-providers-yml \
            '.github/workflows/test-pull-push-providers.yml (pull-push-providers): remaining CI setup and local execution need translation'
        pending ci-test-race-detection-yml \
            '.github/workflows/test-race-detection.yml (race-detection): remaining CI setup and local execution need translation'
        pending ci-test-wsl2-nat-yml \
            '.github/workflows/test-wsl2-nat.yml (wsl2-nat-testpkg, wsl2-nat-testcmd): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
