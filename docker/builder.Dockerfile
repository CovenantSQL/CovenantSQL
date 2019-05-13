# Stage: builder
FROM golang:1.11-alpine3.9 as builder

ARG BUILD_ARG

WORKDIR /go/src/github.com/CovenantSQL/CovenantSQL
COPY . .
RUN apk --no-cache add build-base make git
RUN make clean
RUN GOOS=linux GOLDFLAGS="-linkmode external -extldflags -static" make ${BUILD_ARG}

