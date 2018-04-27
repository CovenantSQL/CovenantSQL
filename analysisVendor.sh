#!/bin/sh

dep status -dot | dot -Tpng -o foo.png && open foo.png
