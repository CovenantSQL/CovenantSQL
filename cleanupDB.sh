#!/bin/sh

PROJECT_DIR=$(cd $(dirname $0)/; pwd)

cd ${PROJECT_DIR} && find . -name '*.db' -exec rm -vf {} \;
cd ${PROJECT_DIR} && find . -name '*.db-shm' -exec rm -vf {} \;
cd ${PROJECT_DIR} && find . -name '*.db-wal' -exec rm -vf {} \;
cd ${PROJECT_DIR} && find . -name 'db.meta' -exec rm -vf {} \;
cd ${PROJECT_DIR} && find . -name 'public.keystore' -exec rm -vf {} \;
cd ${PROJECT_DIR} && find . -name '*.public.keystore' -exec rm -vf {} \;
cd ${PROJECT_DIR} && find . -type d -name '*.ldb' -prune -exec rm -vrf {} \;
