# Stage: builder
# Keep the Alpine version the same as in Dockerfile: the binaries link ICU
# statically but read its data from the runtime image.
FROM golang:1.27-alpine3.24 as builder

RUN apk add --no-cache build-base make git icu-dev icu-static

ARG BUILD_ARG

WORKDIR /go/src/github.com/CovenantSQL/CovenantSQL
COPY . .
RUN make clean
RUN GOOS=linux make ${BUILD_ARG}
