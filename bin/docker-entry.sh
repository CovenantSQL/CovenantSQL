#!/bin/sh

case "${COVENANT_ROLE}" in
miner)
    exec /app/thunderminerd -config "${COVENANT_CONF}"
    ;;
blockproducer)
    exec /app/thunderdbd -config "${COVENANT_CONF}"
    ;;
esac

