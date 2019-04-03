#!/bin/sh

# echo nameserver 114.114.114.114 > /etc/resolv.conf

[ -s "${COVENANT_ALERT}" ] && [ -x "${COVENANT_ALERT}" ] && (eval "${COVENANT_ALERT}")

case "${COVENANT_ROLE}" in
miner)
    exec /app/cql-minerd -config "${COVENANT_CONF}" -metric-web "${METRIC_WEB_ADDR}" "${@}"
    ;;
blockproducer)
    exec /app/cqld -config "${COVENANT_CONF}" -metric-web "${METRIC_WEB_ADDR}" "${@}"
    ;;
observer)
    exec /app/cql explorer -config "${COVENANT_CONF}" -no-password "${COVENANTSQL_OBSERVER_ADDR}" "${@}"
    ;;
adapter)
    exec /app/cql adapter -config "${COVENANT_CONF}" -no-password "${COVENANTSQL_ADAPTER_ADDR}" "${@}"
    ;;
mysql-adapter)
    exec /app/cql-mysql-adapter -config "${COVENANT_CONF}" "${@}"
    ;;
cli)
    exec /app/cql console -config ${COVENANT_CONF} -no-password "${@}"
    ;;
faucet)
    exec /app/cql-faucet -config ${COVENANT_CONF} "${@}"
    ;;
esac

