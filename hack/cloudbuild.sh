#!/usr/bin/env bash
set -o errexit
set -o nounset

# with nounset, these will fail if necessary vars are missing
echo "GIT_TAG: ${GIT_TAG}"
echo "PULL_BASE_REF: ${PULL_BASE_REF}"
echo "PLATFORM: ${PLATFORM}"

# debug the rest of the script in case of image/CI build issues
set -o xtrace

REPO="gcr.io/k8s-staging-sig-storage"
SAMPLE_DRIVER_IMAGE="${REPO}/cosi-driver-sample"

# args to 'make build'
export DOCKER="/buildx-entrypoint" # available in gcr.io/k8s-testimages/gcb-docker-gcloud image
export BUILD_ARGS="--push"
export PLATFORM
export SAMPLE_DRIVER_TAG="${SAMPLE_DRIVER_IMAGE}:${GIT_TAG}"

# build in parallel
make --jobs --output-sync build

# add latest tag to just-built images
gcloud container images add-tag "${SAMPLE_DRIVER_TAG}" "${SAMPLE_DRIVER_IMAGE}:latest"

# PULL_BASE_REF is 'sidecar/TAG' for a controller release
TAG="${PULL_BASE_REF}"
gcloud container images add-tag "${SAMPLE_DRIVER_TAG}" "${SAMPLE_DRIVER_IMAGE}:${TAG}"
