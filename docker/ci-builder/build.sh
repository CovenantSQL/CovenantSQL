#!/bin/bash

docker build -t covenantsql/build:latest . && \
docker push covenantsql/build
