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
            '.github/workflows/ci.yaml (base-lint, base-unit-tests, e2e-docker-short, e2e-hypervisor-short, e2e-iso, e2e-qemu-short, grype-scan, integration-airgapped, integration-aws-amd64, integration-aws-arm64, integration-aws-nvidia-nonfree-lts-amd64, integration-aws-nvidia-nonfree-lts-arm64, integration-aws-nvidia-nonfree-production-amd64, integration-aws-nvidia-nonfree-production-arm64, integration-aws-nvidia-oss-lts-amd64, integration-aws-nvidia-oss-lts-arm64, integration-aws-nvidia-oss-new-feature-amd64, integration-aws-nvidia-oss-new-feature-arm64, integration-aws-nvidia-oss-production-amd64, integration-aws-nvidia-oss-production-arm64, integration-build-aws-image, integration-build-enforcing, integration-build-image-cache, integration-cilium, integration-cloud-images, integration-conformance-default, integration-conformance-enforcing, integration-embedded, integration-extensions, integration-gcp, integration-hypervisor, integration-image-cache, integration-image-factory, integration-images, integration-misc-0, integration-misc-1, integration-misc-2, integration-misc-3, integration-misc-4, integration-provision-0, integration-provision-1, integration-provision-2, integration-provision-3, integration-qemu-13, integration-qemu-csi-longhorn, integration-qemu-csi-netapp, integration-qemu-csi-openebs, integration-qemu-csi-rook-ceph, integration-qemu-default, integration-qemu-encrypted-vip, integration-qemu-enforcing, integration-qemu-ip-family-dualstack, integration-qemu-ip-family-ipv4, integration-qemu-md-boot, integration-qemu-race, integration-reproducibility-test, integration-talosctl, integration-trusted-boot-default, integration-trusted-boot-disk-image, integration-trusted-boot-enforcing, release-metadata-check): remaining CI setup and local execution need translation'
        finish
        ;;
    *)
        echo "Unknown integration action: $1" >&2
        exit 2
        ;;
esac
