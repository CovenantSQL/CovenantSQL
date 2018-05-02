#!/bin/sh

dep ensure
dep status -dot | dot -Tpng -o analysisVendor.png && open analysisVendor.png
