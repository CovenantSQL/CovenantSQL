# Stage: builder
FROM golang:1.10.3-stretch as builder

WORKDIR /go/src/gitlab.com/thunderdb/thunderdb
COPY . .
RUN ./build.sh

# Stage: runner
FROM alpine:3.7

ARG COMMIT
ARG VERSION

LABEL \
    name="covenantsql" \
    homepage="https://gitlab.com/thunderdb/thunderdb.git" \
    maintainer="qi.xiao@covenantsql.io" \
    authors="CovenantSQL Team" \
    commit="${COMMIT}" \
    version="${VERSION}"

ENV VERSION=${VERSION}
ENV COVENANT_ROLE=miner
ENV COVENANT_CONF=config.yaml

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /go/src/gitlab.com/thunderdb/thunderdb/bin/* /app/
ENTRYPOINT [ "./docker-entry.sh" ]
EXPOSE 4661
