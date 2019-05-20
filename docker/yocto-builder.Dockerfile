# Stage: builder
FROM ubuntu:14.04

COPY ./docker/scripts /root/scripts
COPY ./docker/meta-covenantsql /root/meta-covenantsql
RUN /root/scripts/init.sh

USER work
WORKDIR /home/work
RUN ./build.sh

CMD "/bin/bash"
