#!/bin/sh

echo nameserver 1.1.1.1 > /etc/resolv.conf

case "${COVENANT_ROLE}" in
miner)
    exec /app/cql-minerd -config "${COVENANT_CONF}" "${@}"
    ;;
blockproducer)
    exec /app/cqld -config "${COVENANT_CONF}" "${@}"
    ;;
observer)
    exec /app/cql-observer -config "${COVENANT_CONF}" "${@}"
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

