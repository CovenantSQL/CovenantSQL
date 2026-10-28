# Stage: builder
FROM golang:1.27-alpine as builder

RUN apk add --no-cache build-base make git icu-dev icu-static

# musl >= 1.2.4 only declares the LFS64 interfaces (pread64, off64_t, ...) used by
# the bundled sqlite3 with _LARGEFILE64_SOURCE.
ENV CGO_CFLAGS="-O2 -g -D_LARGEFILE64_SOURCE"

ARG BUILD_ARG

WORKDIR /go/src/github.com/CovenantSQL/CovenantSQL
COPY . .
RUN make clean
RUN GOOS=linux make ${BUILD_ARG}
