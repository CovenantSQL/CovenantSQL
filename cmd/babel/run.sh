#! /bin/sh

go build
if [ -f "babel.db" ]
then 
    rm babel.db
fi 
./babel
