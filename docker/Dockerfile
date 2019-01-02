# Stage: all runner
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
COPY --from=covenantsql/covenantsql-builder /go/src/github.com/CovenantSQL/CovenantSQL/bin/* /app/
ENTRYPOINT [ "./docker-entry.sh" ]
EXPOSE 4661
