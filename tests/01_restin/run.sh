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
# We need to add server process here + we need to register ubftab (test.fd)
#
xadmin provision -d \
        -vusv1_name=testsv \
        -vusv1=y \
        -vusv1_sysopt='-e ${NDRX_APPHOME}/log/testsv.log -r' \
        -vaddubf=test.fd

# Add resources
ln -s ../src/testsv/testsv bin/testsv
ln -s ../src/ubftab/test.fd ubftab/test.fd

cd conf

. settest1

xadmin start -y

xadmin stop -c -y

return 0
