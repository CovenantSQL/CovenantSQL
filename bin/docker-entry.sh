#!/bin/sh

echo nameserver 1.1.1.1 > /etc/resolv.conf

case "${COVENANT_ROLE}" in
miner)
    exec /app/cql-minerd -config "${COVENANT_CONF}" "${@}"
    ;;
blockproducer)
    rm -f /app/node_*/chain.db
    exec /app/cqld -config "${COVENANT_CONF}" "${@}"
    ;;
observer)
    rm -f /app/node_observer/observer.db
    MAGIC_DOLLAR='$' envsubst < /etc/nginx/conf.d/servers/explorer.conf.template > /etc/nginx/conf.d/default.conf
    nginx -g 'daemon off;' </dev/null &
    exec /app/cql-observer -config "${COVENANT_CONF}" -listen "${COVENANTSQL_OBSERVER_ADDR}"
    ;;
adapter)
    exec /app/cql-adapter -config "${COVENANT_CONF}" "${@}"
    ;;
mysql-adapter)
    exec /app/cql-mysql-adapter -config "${COVENANT_CONF}" "${@}"
    ;;
cli)
    exec /app/cql -config ${COVENANT_CONF} "${@}"
    ;;
faucet)
    exec /app/cql-faucet -config ${COVENANT_CONF} "${@}"
    ;;
explorer)
    exec /app/cql-explorer -config ${COVENANT_CONF} "${@}"
    ;;
esac

