#! /usr/bin/env bash
set -eo pipefail

export LC_ALL=en_US.UTF-8
export LANG=en_US.UTF-8
export LANGUAGE=en_US.UTF-8
export DISTRO=fsl-imx-wayland
export MACHINE=imx7ulpevk

main() {
    local workspace=/home/work/yocto
    mkdir -p "$workspace"
    cd "$workspace"

    repo init -u https://source.codeaurora.org/external/imx/imx-manifest -b imx-linux-rocko -m imx-4.9.88-2.0.0_ga.xml
    repo sync

    mkdir -p "$workspace/build/conf"
    sed -i 's/^EULA_ACCEPTED=$/EULA_ACCEPTED=1/' "$workspace/sources/base/setup-environment"
    source ./fsl-setup-release.sh

    mv /home/work/meta-covenantsql "$workspace/sources/meta-covenantsql"
    echo 'ACCEPT_FSL_EULA = "1"' >>"$workspace/build/conf/local.conf"
    echo 'BBLAYERS += " ${BSPDIR}/sources/meta-covenantsql"' >>"$workspace/build/conf/bblayers.conf"
    echo 'IMAGE_INSTALL_append = "covenantsql"' >>"$workspace/build/conf/local.conf"
    bitbake core-image-base
}

main "$@"
