# Stage: observer
FROM covenantsql/explorer

ARG COMMIT
ARG VERSION

LABEL \
    name="covenantsql_observer" \
    homepage="https://github.com/CovenantSQL/CovenantSQL.git" \
    maintainer="jin.xu@covenantsql.io" \
    authors="CovenantSQL Team" \
    commit="${COMMIT}" \
    version="${VERSION}"

ENV VERSION=${VERSION}
ENV COVENANT_ROLE=observer
ENV COVENANT_CONF=config.yaml

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=covenantsql/covenantsql-builder /go/src/github.com/CovenantSQL/CovenantSQL/bin/cql-observer /app/
COPY --from=covenantsql/covenantsql-builder /go/src/github.com/CovenantSQL/CovenantSQL/bin/cql /app/
COPY --from=covenantsql/covenantsql-builder /go/src/github.com/CovenantSQL/CovenantSQL/bin/cql-utils /app/
COPY --from=covenantsql/covenantsql-builder /go/src/github.com/CovenantSQL/CovenantSQL/bin/docker-entry.sh /app/
ENTRYPOINT [ "./docker-entry.sh" ]
EXPOSE 4661
EXPOSE 80
