#!/bin/sh

find . -name '*.db' -exec rm -f {} \;
find . -name '*.db-shm' -exec rm -f {} \;
find . -name '*.db-wal' -exec rm -f {} \;
find . -name 'db.meta' -exec rm -f {} \;
find . -name 'public.keystore' -exec rm -f {} \;
find . -name '*.public.keystore' -exec rm -f {} \;
