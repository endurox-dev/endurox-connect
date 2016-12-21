#!/bin/bash

#
# @(#) Test 01 - Rest-IN interface tests...
#

rm -rf ./tmp 2>/dev/null

pushd .

mkdir tmp

cd tmp

#
# Create the env..
#

#
# So we need to add some demo server
#
xadmin provision -d 


cd conf

. settest1

xadmin start -y

xadmin stop -c -y

return 0
