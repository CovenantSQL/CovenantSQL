# Stage: builder
FROM golang:1.11-stretch as builder

WORKDIR /go/src/github.com/CovenantSQL/CovenantSQL
COPY . .
RUN GOOS=linux GOLDFLAGS="-linkmode external -extldflags -static" ./build.sh
RUN rm -f bin/*.test

# Stage: runner
FROM frolvlad/alpine-glibc:latest

ARG COMMIT
ARG VERSION

LABEL \
    name="covenantsql" \
    homepage="https://github.com/CovenantSQL/CovenantSQL.git" \
    maintainer="qi.xiao@covenantsql.io" \
    authors="CovenantSQL Team" \
    commit="${COMMIT}" \
    version="${VERSION}"

ENV VERSION=${VERSION}
ENV COVENANT_ROLE=miner
ENV COVENANT_CONF=config.yaml

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /go/src/github.com/CovenantSQL/CovenantSQL/bin/* /app/
ENTRYPOINT [ "./docker-entry.sh" ]
EXPOSE 4661
