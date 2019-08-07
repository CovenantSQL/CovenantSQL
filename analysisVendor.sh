#!/bin/sh

make dep
dep status -dot | dot -Tpng -o analysisVendor.png && open analysisVendor.png
