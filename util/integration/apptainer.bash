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
        pending ci-ci-yml \
            '.github/workflows/ci.yml (check_go_mod, lint_markdown, check_source, shellcheck, alpine, check_test_corpus, check_license_dependencies, debbuild-debian12, debbuild-debian13, debbuild-ubuntu22, debbuild-ubuntu24, debbuild-ubuntu26, rpmbuild-rocky8, rpmbuild-rocky9, rpmbuild-rocky10, rpmbuild-fedora44, rpmbuild-fedora45, rpmbuild-opensuse-leap, rpmbuild-opensuse-tumbleweed, unit_tests, integration_tests, e2e_tests, check_pkg_no_buildcfg): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
