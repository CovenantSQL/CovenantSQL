#! /bin/sed -nf

# Usage:
#
# Write to new file:
#   ./fmtGoDoc.sed path/to/file/xxx.go >xxx_fmt.go
#
# Or direct overwrite:
#   ./fmtGoDoc.sed -i path/to/files/*.go

/^\/\/ .*$/ b init
p
b

# Initialize godoc block
:init
h
b loop

# Continue godoc block
:next
H
b loop

# Pull next line and branch to next/exit/default
:loop
n
/^\/\/ .*$/ b next
/^\w\+.*$/ b exit
x
p
x
p
b

# Exiting godoc block
:exit
x
s/^\(.*[^.]\)$/\1\./
p
x
p
d
