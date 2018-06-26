#!/bin/bash -x
set -e

../bin/thunderdbd -nodeOffset 0
../bin/thunderdbd -nodeOffset 1
../bin/thunderdbd -nodeOffset 2
../bin/thunderdbd -operation write -client "create table test (test string)"
../bin/thunderdbd -operation write -client "insert into test values(2)"
../bin/thunderdbd -operation read  -client "select * from test"