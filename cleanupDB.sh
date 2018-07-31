#!/bin/sh

find . -name '*.db' -exec rm {} \;
find . -name '*.db-shm' -exec rm {} \;
find . -name '*.db-wal' -exec rm {} \;
find . -name 'db.meta' -exec rm {} \;
find . -name 'public.keystore' -exec rm {} \;
