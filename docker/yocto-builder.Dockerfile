# Stage: builder
FROM ubuntu:14.04

COPY ./docker/scripts /root/scripts
RUN /root/scripts/init.sh

USER work
WORKDIR /home/work
ENV LC_ALL=en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US.UTF-8
ENV DISTRO=fsl-imx-wayland
ENV MACHINE=imx7ulpevk
RUN ./build.sh

CMD "/bin/bash"
