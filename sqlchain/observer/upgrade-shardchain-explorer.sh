#!/usr/bin/env bash
#
# This script is used for upgrading the embeded shard chain explorer.

set -o errexit
set -o pipefail
set -o nounset

ROOT_DIR=$( cd "$(dirname "${BASH_SOURCE[0]}")" && pwd )

env::has_command() { command -v "$1" >/dev/null 2>&1; }

env::ensure_command() {
    for opt_command in "$@"; do
        if env::has_command "${opt_command}"; then
          echo "${opt_command}"
          return
        fi
    done

    echo "You MUST have one of these command(s) installed: $@"
    exit 1
}

main() {
    CMD_PM="$( env::ensure_command "yarn" "npm" )"
    {
      env::ensure_command "git"
      env::ensure_command "statik"
    } &>/dev/null

    echo "PKG manager tool: ${CMD_PM}"

    tmp_source_dir="$( mktemp -d "${TMPDIR:-/tmp/}$(basename $0).XXXXXX" )"
    echo "Temporary working directory: ${tmp_source_dir}"

    set -o xtrace
    git clone https://github.com/CovenantSQL/shardchain-explorer "${tmp_source_dir}"
    cd "${tmp_source_dir}"
    ${CMD_PM} install
    ${CMD_PM} run build

    statik -src="${tmp_source_dir}/dist" -dest "${ROOT_DIR}/"
}

main "$@"

