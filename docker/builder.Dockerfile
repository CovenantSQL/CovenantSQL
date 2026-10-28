# Stage: builder
FROM golang:1.27-alpine as builder

RUN apk add --no-cache build-base make git icu-dev icu-static

ARG BUILD_ARG

WORKDIR /go/src/github.com/CovenantSQL/CovenantSQL
COPY . .
RUN make clean
RUN GOOS=linux make ${BUILD_ARG}
