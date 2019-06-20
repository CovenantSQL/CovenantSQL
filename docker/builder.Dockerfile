# Stage: builder
FROM covenantsql/release-builder as builder

ARG BUILD_ARG

WORKDIR /go/src/github.com/CovenantSQL/CovenantSQL
COPY . .
RUN make clean
RUN GOOS=linux make ${BUILD_ARG}

