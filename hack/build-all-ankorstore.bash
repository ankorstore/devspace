#!/usr/bin/env bash
# This script will build devspace and calculate hash for each
# (DEVSPACE_BUILD_PLATFORMS, DEVSPACE_BUILD_ARCHS) pair.
# DEVSPACE_BUILD_PLATFORMS="linux" DEVSPACE_BUILD_ARCHS="amd64" ./hack/build-all.bash
# can be called to build only for linux-amd64

set -e

export GO111MODULE=on
export GOFLAGS=-mod=vendor

# Update vendor directory
# go mod vendor

DEVSPACE_ROOT=$(git rev-parse --show-toplevel)
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null)
DATE=$(date "+%Y-%m-%d")

echo "Current working directory is $(pwd)"
echo "PATH is $PATH"
echo "GOPATH is $GOPATH"

if [[ "$(pwd)" != "${DEVSPACE_ROOT}" ]]; then
  echo "you are not in the root of the repo" 1>&2
  echo "please cd to ${DEVSPACE_ROOT} before running this script" 1>&2
  exit 1
fi

GO_BUILD_CMD="go build -a"
GO_BUILD_LDFLAGS="-s -w -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${DATE} -X main.version=${VERSION}"

if [[ -z "${DEVSPACE_BUILD_PLATFORMS}" ]]; then
    DEVSPACE_BUILD_PLATFORMS="linux darwin"
    DEVSPACE_BUILD_PLATFORMS="darwin"
fi

if [[ -z "${DEVSPACE_BUILD_ARCHS}" ]]; then
    DEVSPACE_BUILD_ARCHS="amd64 arm64"
fi

# Create the release directory
mkdir -p "${DEVSPACE_ROOT}/release"

# build devspace helper
echo "Building devspace helper"
GOARCH=amd64 GOOS=linux go build -ldflags "-s -w -X github.com/loft-sh/devspace/helper/cmd.version=${VERSION}" -o "${DEVSPACE_ROOT}/release/devspacehelper" helper/main.go
upx "${DEVSPACE_ROOT}/release/devspacehelper" #compress devspacehelper
shasum -a 256 "${DEVSPACE_ROOT}/release/devspacehelper" > "${DEVSPACE_ROOT}/release/devspacehelper".sha256

GOARCH=arm64 GOOS=linux go build -ldflags "-s -w -X github.com/loft-sh/devspace/helper/cmd.version=${VERSION}" -o "${DEVSPACE_ROOT}/release/devspacehelper-arm64" helper/main.go
upx "${DEVSPACE_ROOT}/release/devspacehelper-arm64" #compress devspacehelper
shasum -a 256 "${DEVSPACE_ROOT}/release/devspacehelper-arm64" > "${DEVSPACE_ROOT}/release/devspacehelper-arm64".sha256

for OS in ${DEVSPACE_BUILD_PLATFORMS[@]}; do
  for ARCH in ${DEVSPACE_BUILD_ARCHS[@]}; do
    NAME="devspace-${OS}-${ARCH}"
    if [[ "${OS}" == "windows" ]]; then
      NAME="${NAME}.exe"
    fi

    # darwin 386 is deprecated and shouldn't be used anymore
    if [[ "${ARCH}" == "386" && "${OS}" == "darwin" ]]; then
        echo "Building for ${OS}/${ARCH} not supported."
        continue
    fi

    # arm64 build is only supported for darwin
    if [[ "${ARCH}" == "arm64" && "${OS}" == "windows" ]]; then
        echo "Building for ${OS}/${ARCH} not supported."
        continue
    fi

    echo "Building for ${OS}/${ARCH}"

    # build darwin with CGO_ENABLED=1
    if [[ "${OS}" == "darwin" ]]; then
      CGO_ENABLED=1
    else
      CGO_ENABLED=0
    fi

    # build the DevSpace binary
    CGO_ENABLED=${CGO_ENABLED} GOARCH=${ARCH} GOOS=${OS} ${GO_BUILD_CMD} -ldflags "${GO_BUILD_LDFLAGS}"\
                  -o "${DEVSPACE_ROOT}/release/${NAME}" .
    shasum -a 256 "${DEVSPACE_ROOT}/release/${NAME}" > "${DEVSPACE_ROOT}/release/${NAME}".sha256
  done
done
