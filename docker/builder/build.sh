#!/bin/bash

docker build -t covenantsql/release-builder:latest . && \
docker push covenantsql/release-builder
