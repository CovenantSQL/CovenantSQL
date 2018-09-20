#!/bin/sh

case "${COVENANT_ROLE}" in
miner)
    exec /app/covenantminerd -config "${COVENANT_CONF}"
    ;;
blockproducer)
    exec /app/covenantsqld -config "${COVENANT_CONF}"
    ;;
observer)
    exec /app/covenantobserver -config "${COVENANT_CONF}" "${@}"
    ;;
adapter)
    exec /app/covenantadapter -config "${COVENANT_CONF}" "${@}"
    ;;
cli)
    exec /app/covenantcli -config ${COVENANT_CONF} "${@}"
    ;;
faucet)
    exec /app/covenantfaucet -config ${COVENANT_CONF} "${@}"
    ;;
esac

