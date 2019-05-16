# Stage: builder
FROM ubuntu:14.04

RUN apt-get update \
    && apt-get install -y apt-transport-https \
    && > /etc/apt/sources.list echo "deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty main restricted universe multiverse" \
    && >> /etc/apt/sources.list echo "deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-updates main restricted universe multiverse" \
    && >> /etc/apt/sources.list echo "deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-backports main restricted universe multiverse" \
    && >> /etc/apt/sources.list echo "deb https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ trusty-security main restricted universe multiverse" \
    && apt-get update \
    && apt-get install -y \
        gawk \
        wget \
        git-core \
        diffstat \
        unzip \
        texinfo \
        gcc-multilib \
        build-essential \
        chrpath \
        socat \
        libsdl1.2-dev \
        xterm \
        sed \
        cvs \
        subversion \
        coreutils \
        texi2html \
        docbook-utils \
        python-pysqlite2 \
        help2man \
        make \
        gcc \
        g++ \
        desktop-file-utils \
        libgl1-mesa-dev \
        libglu1-mesa-dev \
        mercurial \
        autoconf \
        automake \
        groff \
        curl \
        lzop \
        asciidoc \
        u-boot-tools \
        cpio \
        sudo \
        locales \
    # install git-repo
    && curl https://mirrors.tuna.tsinghua.edu.cn/git/git-repo >/usr/bin/repo \
    && chmod a+x /usr/bin/repo \
    # setup work user
    && useradd --create-home work \
    && echo "work ALL=(ALL) NOPASSWD: ALL" >/etc/sudoers \
    # setup locale
    && locale-gen en_US.UTF-8 \
    && update-locale LANG=en_US.UTF-8 LC_ALL=en_US.UTF-8 \
    && dpkg-reconfigure --frontend=noninteractive locales \
    # use bash as default shell
    && ln -fs /bin/bash /bin/sh

ENV LC_ALL=en_US.UTF-8 \
    LANG=en_US.UTF-8 \
    LANGUAGE=en_US.UTF-8 \
    DISTRO=fsl-imx-wayland \
    MACHINE=imx7ulpevk

USER work
WORKDIR /home/work
RUN repo init -u https://source.codeaurora.org/external/imx/imx-manifest -b imx-linux-rocko -m imx-4.9.88-2.0.0_ga.xml \
    && repo sync \
    && mkdir -p build/conf && echo 'ACCEPT_FSL_EULA = "1"' >> build/conf/local.conf \
    && source fsl-setup-release.sh

COPY ./meta-covenantsql /home/work/build/sources/meta-covenantsql
RUN echo 'BBLAYERS += " ${BSPDIR}/sources/meta-covenantsql"' >> build/conf/bblayers.conf \
    && echo 'IMAGE_INSTALL_append = "covenantsql"' >> build/conf/local.conf \
    && bitbake core-image-base

CMD "/bin/bash"
