#! /usr/bin/env bash
set -euo pipefail

config_packages() {
    apt-get update
    apt-get install -y apt-transport-https
    cat >/etc/apt/sources.list <<LIST
# 默认注释了源码镜像以提高 apt update 速度，如有需要可自行取消注释
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty main restricted universe multiverse
# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty main restricted universe multiverse
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-updates main restricted universe multiverse
# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-updates main restricted universe multiverse
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-backports main restricted universe multiverse
# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-backports main restricted universe multiverse
deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-security main restricted universe multiverse
# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-security main restricted universe multiverse

# 预发布软件源，不建议启用
# deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-proposed main restricted universe multiverse
# deb-src https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-proposed main restricted universe multiverse
LIST

    apt-get update
    local packages=(
        gawk
        wget
        git-core
        diffstat
        unzip
        texinfo
        gcc-multilib
        build-essential
        chrpath
        socat
        libsdl1.2-dev
        xterm
        sed
        cvs
        subversion
        coreutils
        texi2html
        docbook-utils
        python-pysqlite2
        help2man
        make
        gcc
        g++
        desktop-file-utils
        libgl1-mesa-dev
        libglu1-mesa-dev
        mercurial
        autoconf
        automake
        groff
        curl
        lzop
        asciidoc
        u-boot-tools
        cpio
        sudo
        locales
    )
    apt-get install -y "${packages[@]}"

    curl -o/usr/bin/repo https://mirrors.tuna.tsinghua.edu.cn/git/git-repo
    chmod +x /usr/bin/repo
}

config_user() {
    local id=work
    useradd --create-home "$id"
    echo "$id ALL=(ALL:ALL) NOPASSWD:ALL" >>/etc/sudoers

    mv /root/scripts/build.sh /home/work/
    chown -R work:work /home/work
}

config_locale() {
    locale-gen en_US.UTF-8
    update-locale LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8
    dpkg-reconfigure --frontend=noninteractive locales
}

main() {
    if [[ $UID != 0 ]]; then
        >&2 echo "Must run as root"
        return 1
    fi
    config_packages
    config_user
    config_locale
}

main "$@"
