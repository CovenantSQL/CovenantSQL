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
esac

