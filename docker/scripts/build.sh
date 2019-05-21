#! /usr/bin/env bash
set -eo pipefail

main() {
    local workspace=/home/work/yocto
    mkdir -p "$workspace"
    cd "$workspace"

    repo init -u https://source.codeaurora.org/external/imx/imx-manifest -b imx-linux-rocko -m imx-4.9.88-2.0.0_ga.xml
    repo sync

    sed -i 's/^EULA_ACCEPTED=$/EULA_ACCEPTED=1/' "$workspace/sources/base/setup-environment"
    source ./fsl-setup-release.sh

    # Patch repo and add covenantsql layer
    git clone https://github.com/CovenantSQL/meta-covenantsql.git "$workspace/sources/meta-covenantsql"
    echo 'ACCEPT_FSL_EULA = "1"' >>"$workspace/build/conf/local.conf"
    echo 'BBLAYERS += " ${BSPDIR}/sources/meta-covenantsql"' >>"$workspace/build/conf/bblayers.conf"
    echo 'IMAGE_INSTALL_append = "cql-minerd cql cqld"' >>"$workspace/build/conf/local.conf"

    bitbake core-image-base
}

main "$@"
