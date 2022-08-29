#!/bin/bash
set -x

OS=$1
ROOT=$(dirname $2)
VERSION=$3
ARCH=$4

if [ $OS = "darwin" ]; then
    mkdir -p macos/${ARCH}
    pkgbuild --quiet \
             --root $ROOT \
             --identifier "com.calyptia.cli" \
             --version $VERSION \
             --install-location "/usr/local/bin" "macos/${ARCH}/calyptia-cli.pkg"

    productbuild --quiet --package macos/${ARCH}/calyptia-cli.pkg calyptia-cli-v${VERSION}-${ARCH}.pkg
    rm -f macos/${ARCH}/calyptia-cli.pkg
fi
